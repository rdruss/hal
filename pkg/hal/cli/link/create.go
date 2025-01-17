package link

import (
	"fmt"
	"github.com/spf13/cobra"
	link "halkyon.io/api/link/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"halkyon.io/hal/pkg/ui"
	k8score "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
)

const (
	targetSeparator = ": "
)

type createOptions struct {
	targetName string
	secret     string
	linkType   link.LinkType
	*cmdutil.CreateOptions
	*cmdutil.EnvOptions
	target *link.Link
}

func (o *createOptions) SetEnvOptions(env *cmdutil.EnvOptions) {
	o.EnvOptions = env
}

func (o *createOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	// first check if proper parameters combination are provided
	useSecret := len(o.secret) > 0
	useEnv := len(o.EnvPairs) > 0
	if useSecret && useEnv {
		return fmt.Errorf("invalid parameter combination: either pass a secret name or environment variables, not both")
	}

	// retrieve and build list of available targets
	capabilitiesAndComponents, validTarget, err := o.checkAndGetValidTargets()
	if err != nil {
		return err
	}
	if len(capabilitiesAndComponents) == 0 {
		return fmt.Errorf("no valid capabilities or components currently exist on the cluster")
	}
	ui.OutputSelection("Selected target", o.targetName)
	if !validTarget {
		o.targetName = o.extractTargetName(ui.Select("Target", capabilitiesAndComponents))
	}

	if !useSecret && !useEnv {
		useSecret = ui.Proceed("Use Secret")
	}

	if useSecret {
		o.linkType = link.SecretLinkType
		ui.OutputSelection("Selected link type", o.linkType.String())
		secrets, valid, err := o.checkAndGetValidSecrets()
		if err != nil {
			return err
		}
		if len(secrets) == 0 {
			return fmt.Errorf("no valid secrets currently exist on the cluster")
		}
		if !valid {
			msg := "Secret (only potential matches shown)"
			if len(o.secret) > 0 {
				msg = ui.SelectFromOtherErrorMessage("Unknown secret", o.secret)
			}
			o.secret = ui.Select(msg, secrets)
		}
	} else {
		o.linkType = link.EnvLinkType
		ui.OutputSelection("Selected link type", o.linkType.String())
		if err := o.EnvOptions.Complete(name, cmd, args); err != nil {
			return err
		}
	}

	return nil
}

func (o *createOptions) Validate() error {
	return nil
}

func (o *createOptions) Build() runtime.Object {
	if o.target == nil {
		o.target = &link.Link{
			ObjectMeta: v1.ObjectMeta{
				Name:      o.Name,
				Namespace: o.CreateOptions.Client.GetNamespace(),
			},
			Spec: link.LinkSpec{
				ComponentName: o.targetName,
				Type:          o.linkType,
				Ref:           o.secret,
				Envs:          o.Envs,
			},
		}
	}
	return o.target
}

func (o *createOptions) GeneratePrefix() string {
	return o.targetName
}

func (o *createOptions) Set(entity runtime.Object) {
	o.target = entity.(*link.Link)
}

func NewCmdCreate(parent string) *cobra.Command {
	c := k8s.GetClient()
	o := &createOptions{}
	generic := cmdutil.NewCreateOptions(cmdutil.Link, client{
		client: c.HalkyonLinkClient.Links(c.Namespace),
		ns:     c.Namespace,
	})
	generic.Delegate = o
	o.CreateOptions = generic
	l := cmdutil.NewGenericCreate(parent, generic)
	l.Long = `Link the current (or target) component to the specified capability or component`
	l.Example = fmt.Sprintf("  # links the client-sb to the backend-sb component\n %s -n client-to-backend -t client-sb", cmdutil.CommandName(l.Name(), parent))

	l.Flags().StringVarP(&o.targetName, "target", "t", "", "Name of the component or capability to link to")
	l.Flags().StringVarP(&o.secret, "secret", "s", "", "Secret name to reference if using Secret type")

	cmdutil.SetupEnvOptions(o, l)

	return l
}

func (o *createOptions) checkAndGetValidTargets() ([]string, bool, error) {
	const capabilityPrefix = "capability"
	const componentPrefix = "component"
	known := make([]string, 0, 10)
	givenIsValid := false

	client := k8s.GetClient()
	capabilities, err := client.HalkyonCapabilityClient.Capabilities(client.Namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, false, err
	}
	for _, c := range capabilities.Items {
		known = append(known, fmt.Sprintf("%s%s%s", capabilityPrefix, targetSeparator, c.Name))
		if !givenIsValid && c.Name == o.targetName {
			givenIsValid = true
		}
	}

	components, err := client.HalkyonComponentClient.Components(client.Namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, false, err
	}
	for _, c := range components.Items {
		known = append(known, fmt.Sprintf("%s%s%s", componentPrefix, targetSeparator, c.Name))
		if !givenIsValid && c.Name == o.targetName {
			givenIsValid = true
		}
	}

	return known, givenIsValid, nil
}

func (createOptions) extractTargetName(typeAndTarget string) string {
	index := strings.Index(typeAndTarget, targetSeparator)
	return typeAndTarget[index+len(targetSeparator):]
}

func (o *createOptions) checkAndGetValidSecrets() ([]string, bool, error) {
	client := k8s.GetClient()
	secrets, err := client.KubeClient.CoreV1().Secrets(client.Namespace).List(v1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("type", string(k8score.SecretTypeOpaque)).String(),
	})
	if err != nil {
		return nil, false, err
	}
	known := make([]string, 0, len(secrets.Items))
	givenIsValid := false
	for _, secret := range secrets.Items {
		known = append(known, secret.Name)
		if secret.Name == o.secret {
			givenIsValid = true
		}
	}
	return known, givenIsValid, nil
}
