// +build integration

package volumes_test

import (
	"context"
	"crypto/md5"
	"fmt"
	"testing"

	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/executors/docker/internal/labels"
	"gitlab.com/gitlab-org/gitlab-runner/executors/docker/internal/volumes"
	"gitlab.com/gitlab-org/gitlab-runner/executors/docker/internal/volumes/parser"
	"gitlab.com/gitlab-org/gitlab-runner/helpers"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/docker"
)

func testCreateVolumesLabels(t *testing.T, p parser.Parser) {
	helpers.SkipIntegrationTests(t, "docker", "info")

	successfulJobResponse, err := common.GetRemoteSuccessfulBuild()
	require.NoError(t, err)

	client, err := docker.New(docker.Credentials{}, "")
	require.NoError(t, err, "should be able to connect to docker")
	defer client.Close()

	successfulJobResponse.GitInfo.RepoURL = "https://user:pass@gitlab.example.com/namespace/project.git"

	build := &common.Build{
		ProjectRunnerID: 0,
		Runner: &common.RunnerConfig{
			RunnerCredentials: common.RunnerCredentials{Token: "test-token"},
		},
		JobResponse: successfulJobResponse,
	}
	build.Variables = common.JobVariables{
		{Key: "CI_PIPELINE_ID", Value: "1"},
	}

	logger, _ := logrustest.NewNullLogger()

	cfg := volumes.ManagerConfig{
		CacheDir:     "",
		BasePath:     "",
		UniqueName:   t.Name(),
		DisableCache: false,
	}

	manager := volumes.NewManager(logger, p, client, cfg, labels.NewLabeler(build))

	ctx := context.Background()

	err = manager.Create(ctx, testCreateVolumesLabelsDestinationPath)
	assert.NoError(t, err)

	name := fmt.Sprintf("%s-cache-%x", t.Name(), md5.Sum([]byte(testCreateVolumesLabelsDestinationPath)))
	defer func() {
		err = client.VolumeRemove(ctx, name, true)
		assert.NoError(t, err)
	}()

	volume, err := client.VolumeInspect(ctx, name)
	require.NoError(t, err)

	assert.Equal(t, map[string]string{
		"com.gitlab.gitlab-runner.job.before_sha":  "ca50079dac5293292f83a4d454922ba8db44e7a3",
		"com.gitlab.gitlab-runner.job.id":          "0",
		"com.gitlab.gitlab-runner.job.url":         "https://gitlab.example.com/namespace/project/-/jobs/0",
		"com.gitlab.gitlab-runner.job.ref":         "master",
		"com.gitlab.gitlab-runner.job.sha":         "91956efe32fb7bef54f378d90c9bd74c19025872",
		"com.gitlab.gitlab-runner.managed":         "true",
		"com.gitlab.gitlab-runner.pipeline.id":     "1",
		"com.gitlab.gitlab-runner.project.id":      "0",
		"com.gitlab.gitlab-runner.runner.id":       "test-tok",
		"com.gitlab.gitlab-runner.runner.local_id": "0",
		"com.gitlab.gitlab-runner.type":            "cache",
	}, volume.Labels)
}
