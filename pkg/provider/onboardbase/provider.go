package onboardbase

import (
	"context"
	"fmt"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	oClient "github.com/external-secrets/external-secrets/pkg/provider/onboardbase/client"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	errNewClient        = "unable to create OnboardbaseClient : %s"
	errInvalidStore     = "invalid store: %s"
	errOnboardbaseStore = "missing or invalid Onboardbase SecretStore"
)

// Provider is a Onboardbase secrets provider implementing NewClient and ValidateStore for the esv1beta1.Provider interface.
type Provider struct{}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &Client{}
var _ esv1beta1.Provider = &Provider{}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Onboardbase: &esv1beta1.OnboardbaseProvider{},
	})
}

func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()

	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Onboardbase == nil {
		return nil, fmt.Errorf(errOnboardbaseStore)
	}

	onboardbaseStoreSpec := storeSpec.Provider.Onboardbase

	client := &Client{
		kube:      kube,
		store:     onboardbaseStoreSpec,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}

	if err := client.setAuth(ctx); err != nil {
		return nil, err
	}

	onboardbase, err := oClient.NewOnboardbaseClient(client.onboardbaseAPIKey, client.onboardbasePasscode)
	if err != nil {
		return nil, fmt.Errorf(errNewClient, err)
	}

	client.onboardbase = onboardbase
	client.project = client.store.Project
	client.environment = client.store.Environment

	return client, nil
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	storeSpec := store.GetSpec()
	onboardbaseStoreSpec := storeSpec.Provider.Onboardbase
	onboardbaseAPIKeySecretRef := onboardbaseStoreSpec.Auth.OnboardbaseAPIKey
	if err := utils.ValidateSecretSelector(store, onboardbaseAPIKeySecretRef); err != nil {
		return fmt.Errorf(errInvalidStore, err)
	}

	if onboardbaseAPIKeySecretRef.Name == "" {
		return fmt.Errorf(errInvalidStore, "onboardbaseAPIKey.name cannot be empty")
	}

	onboardbasePasscodeKeySecretRef := onboardbaseStoreSpec.Auth.OnboardbasePasscode
	if err := utils.ValidateSecretSelector(store, onboardbasePasscodeKeySecretRef); err != nil {
		return fmt.Errorf(errInvalidStore, err)
	}

	if onboardbasePasscodeKeySecretRef.Name == "" {
		return fmt.Errorf(errInvalidStore, "onboardbasePasscode.name cannot be empty")
	}

	return nil
}
