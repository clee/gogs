// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dchest/scrypt"
)

// User types.
const (
	UT_INDIVIDUAL = iota + 1
	UT_ORGANIZATION
)

// Login types.
const (
	LT_PLAIN = iota + 1
	LT_LDAP
)

// A User represents the object of individual and member of organization.
type User struct {
	Id            int64
	LowerName     string `xorm:"unique not null"`
	Name          string `xorm:"unique not null"`
	Email         string `xorm:"unique not null"`
	Passwd        string `xorm:"not null"`
	LoginType     int
	Type          int
	NumFollowers  int
	NumFollowings int
	NumStars      int
	NumRepos      int
	Avatar        string    `xorm:"varchar(2048) not null"`
	Created       time.Time `xorm:"created"`
	Updated       time.Time `xorm:"updated"`
}

// A Follow represents
type Follow struct {
	Id       int64
	UserId   int64     `xorm:"unique(s)"`
	FollowId int64     `xorm:"unique(s)"`
	Created  time.Time `xorm:"created"`
}

// Operation types of repository.
const (
	OP_CREATE_REPO = iota + 1
	OP_DELETE_REPO
	OP_STAR_REPO
	OP_FOLLOW_REPO
	OP_COMMIT_REPO
	OP_PULL_REQUEST
)

// A Action represents
type Action struct {
	Id      int64
	UserId  int64
	OpType  int
	RepoId  int64
	Content string
	Created time.Time `xorm:"created"`
}

var (
	ErrUserAlreadyExist = errors.New("User already exist")
	ErrUserNotExist     = errors.New("User does not exist")
)

// IsUserExist checks if given user name exist,
// the user name should be noncased unique.
func IsUserExist(name string) (bool, error) {
	return orm.Get(&User{LowerName: strings.ToLower(name)})
}

// validateUser checks if user exist.
func validateUser(name string) error {
	isExist, err := IsUserExist(name)
	if err != nil {
		return err
	} else if isExist {
		return ErrUserAlreadyExist
	}
	return nil
}

// RegisterUser creates record of a new user.
func RegisterUser(user *User) (err error) {
	if err = validateUser(user.Name); err != nil {
		return err
	}
	_, err = orm.Insert(user)
	return err
}

// UpdateUser updates user's information.
func UpdateUser(user *User) (err error) {
	_, err = orm.Id(user.Id).Update(user)
	return err
}

// DeleteUser completely deletes everything of the user.
func DeleteUser(user *User) error {
	// TODO: check if has ownership of any repository.
	_, err := orm.Delete(user)
	// TODO: delete and update follower information.
	return err
}

// EncodePasswd encodes password to safe format.
func (user *User) EncodePasswd(pass string) error {
	newPasswd, err := scrypt.Key([]byte(user.Passwd), []byte("!#@FDEWREWR&*("), 16384, 8, 1, 64)
	user.Passwd = fmt.Sprintf("%x", newPasswd)
	return err
}

// LoginUserPlain validates user by raw user name and password.
func LoginUserPlain(name, passwd string) (*User, error) {
	user := User{Name: name}
	if err := user.EncodePasswd(passwd); err != nil {
		return nil, err
	}

	has, err := orm.Get(&user)
	if !has {
		err = ErrUserNotExist
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FollowUser marks someone be another's follower.
func FollowUser(userId int64, followId int64) error {
	session := orm.NewSession()
	defer session.Close()
	session.Begin()
	_, err := session.Insert(&Follow{UserId: userId, FollowId: followId})
	if err != nil {
		session.Rollback()
		return err
	}
	_, err = session.Exec("update user set num_followers = num_followers + 1 where id = ?", followId)
	if err != nil {
		session.Rollback()
		return err
	}
	_, err = session.Exec("update user set num_followings = num_followings + 1 where id = ?", userId)
	if err != nil {
		session.Rollback()
		return err
	}
	return session.Commit()
}

// UnFollowUser unmarks someone be another's follower.
func UnFollowUser(userId int64, unFollowId int64) error {
	session := orm.NewSession()
	defer session.Close()
	session.Begin()
	_, err := session.Delete(&Follow{UserId: userId, FollowId: unFollowId})
	if err != nil {
		session.Rollback()
		return err
	}
	_, err = session.Exec("update user set num_followers = num_followers - 1 where id = ?", unFollowId)
	if err != nil {
		session.Rollback()
		return err
	}
	_, err = session.Exec("update user set num_followings = num_followings - 1 where id = ?", userId)
	if err != nil {
		session.Rollback()
		return err
	}
	return session.Commit()
}