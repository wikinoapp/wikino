package viewmodel

import "github.com/wikinoapp/wikino/go/internal/model"

// Profileはプロフィールの画面に表示するユーザーの情報です
type Profile struct {
	Atname      string
	Name        string
	Description string
}

// NewProfileはユーザーのモデルからProfileを生成します
func NewProfile(user *model.User) Profile {
	return Profile{
		Atname:      user.Atname,
		Name:        user.Name,
		Description: user.Description,
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
