package remote

type providerSt struct {
	provider      string
	endpoint      string
	path          string
	secretKeyring string
}

func (rp providerSt) Provider() string {
	return rp.provider
}

func (rp providerSt) Endpoint() string {
	return rp.endpoint
}

func (rp providerSt) Path() string {
	return rp.path
}

func (rp providerSt) SecretKeyring() string {
	return rp.secretKeyring
}

func NewProviderSt(provider, endpoint, path, secretKeyring string) *providerSt {
	return &providerSt{
		provider:      provider,
		endpoint:      endpoint,
		path:          path,
		secretKeyring: secretKeyring,
	}
}
