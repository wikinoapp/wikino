package viewmodel

import (
	"strings"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// Profileはプロフィールの画面に表示するユーザーの情報です
type Profile struct {
	Atname      string
	Name        string
	Description string
}

// NewProfileは空白だけの名前と自己紹介を未設定にしたProfileを生成します。
func NewProfile(user *model.User) Profile {
	name := user.Name
	if strings.TrimSpace(name) == "" {
		name = ""
	}
	description := user.Description
	if strings.TrimSpace(description) == "" {
		description = ""
	}

	return Profile{
		Atname:      user.Atname,
		Name:        name,
		Description: description,
	}
}

// Titleは見出しに出す名前を返します。名前が空ならアットネームを出します。
func (p Profile) Title() string {
	if p.Name == "" {
		return "@" + p.Atname
	}
	return p.Name
}

// Subtitleは見出しの下に出すアットネームを返します。見出しにアットネームを出すときは
// 重ねて出さないよう空を返します。
func (p Profile) Subtitle() string {
	if p.Name == "" {
		return ""
	}
	return "@" + p.Atname
}
