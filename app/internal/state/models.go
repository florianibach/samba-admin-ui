package state

type User struct {
	Name string
	UID  *int
	GID  *int
}

type Group struct {
	Name string
	GID  *int
}

type Membership struct {
	User  string
	Group string
}
