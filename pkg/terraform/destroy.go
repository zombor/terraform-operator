package terraform

import (
	"bytes"
	"context"

	"github.com/hashicorp/go-version"
	install "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hc-install/src"
	"github.com/hashicorp/terraform-exec/tfexec"
)

func Destroy(path string) (string, error) {
	i := install.NewInstaller()
	execPath, err := i.Ensure(context.Background(), []src.Source{
		&releases.ExactVersion{
			Product: product.Terraform,
			Version: version.Must(version.NewVersion("1.4.0")),
		},
	})
	if err != nil {
		return "", err
	}

	defer i.Remove(context.Background())

	tf, err := tfexec.NewTerraform(path, execPath)
	if err != nil {
		return "", err
	}

	var buff bytes.Buffer

	tf.SetStdout(&buff)

	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		return "", err
	}

	err = tf.Destroy(context.Background())

	return buff.String(), err
}
