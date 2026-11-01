package store


type AuthInfo struct {
	// true if the token is authorized
	IsAuthorized bool
	// Name of the key's environment
	Environment string
	// Name of the key's source
	Source string
}


// The purpose of this object is to act as middleware between key store and clients.
// There are 2 types of clients:
// - Auth client that requires fast in-memory lookups to authorize keys via api
// - Admin client to modify the actual stored data
// Why this middleware is needed is because there is both a memory store and an
// in-memory structure.
type Registry struct {

}

func (r *Registry) Authorize(token string) (authInfo AuthInfo, err error) {
	return AuthInfo{}, nil
}

func (r *Registry) List(count int, offset int) (keyViews []KeyView, err error) {
	return nil, nil
}

func (r *Registry) Delete(keyID uint) (err error) {
	return nil
}

func (r *Registry) Create(keyView KeyView) (createdKeyView KeyView, err error) {
	return KeyView{}, nil
}

func (r *Registry) Update(keyView KeyView) (err error) {
	return nil
}
