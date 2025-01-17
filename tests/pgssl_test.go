package tests

import (
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

func TestPgSSL(t *testing.T) {
	t.Parallel()

	var (
		secretName = pointer.String("pg-certs")
	)

	objs := LoadChart(t).MustRender(t, func(cv *CoderValues) {
		cv.Postgres.Default.Enable = pointer.Bool(false)

		cv.Postgres.Host = pointer.String("1.1.1.1")
		cv.Postgres.Port = pointer.String("5432")
		cv.Postgres.User = pointer.String("postgres")

		cv.Postgres.Database = pointer.String("postgres")
		cv.Postgres.PasswordSecret = pointer.String("pg-pass")

		cv.Postgres.SSLMode = pointer.String("require")
		cv.Postgres.SSL.CertSecret.Name = secretName
		cv.Postgres.SSL.CertSecret.Key = pointer.String("cert")
		cv.Postgres.SSL.KeySecret.Name = secretName
		cv.Postgres.SSL.KeySecret.Key = pointer.String("cert")
		cv.Postgres.SSL.RootCertSecret.Name = secretName
		cv.Postgres.SSL.RootCertSecret.Key = pointer.String("cert")
	})
	coderd := MustFindDeployment(t, objs, "coderd")

	for _, vol := range []string{"pgcert", "pgkey", "pgrootcert"} {
		AssertVolume(t, coderd.Spec.Template.Spec.Volumes, vol, func(t testing.TB, v corev1.Volume) {
			require.NotNilf(t, v.Secret, "secret nil for %q", vol)
			assert.Equalf(t, "pg-certs", v.Secret.SecretName, "secret name incorrect for %q", vol)
		})
	}

	for _, cnt := range []string{"migrations", "coderd"} {
		// Combine both init and regular containers.
		cnts := append(coderd.Spec.Template.Spec.InitContainers, coderd.Spec.Template.Spec.Containers...)

		AssertContainer(t, cnts, cnt, func(t testing.TB, c corev1.Container) {
			for _, vol := range []string{"pgcert", "pgkey", "pgrootcert"} {
				AssertVolumeMount(t, c.VolumeMounts, vol, func(t testing.TB, v corev1.VolumeMount) {
					assert.Equalf(t, vol, v.Name, "volume mount name incorrect for %q", vol)
					assert.Truef(t, v.ReadOnly, "readonly incorrect for %q", vol)
					assert.Equalf(t, path.Join("/etc/ssl/certs/pg/", strings.TrimPrefix(v.Name, "pg")), v.MountPath, "mount path incorrect for %q", vol)
				})
			}
		})
	}
}
