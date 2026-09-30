package kea

func Resolve(url, user, password string) (*Client, error) {
	if url == "" {
		return LoadLocal()
	}

	return New(url, user, password), nil
}
