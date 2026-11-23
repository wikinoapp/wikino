package validator

import (
	"context"
	"regexp"
	"slices"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// アットネームのバリデーション定数
const (
	AtnameMaxLength = 20
)

// アットネームの形式 (英数字とアンダースコアのみ)
var atnameRegex = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// reservedAtnamesはユーザーが使えないアットネーム。サービスの運営者や、URLの一部と
// 紛らわしい名前を名乗れないようにする。
var reservedAtnames = []string{
	"admin",
	"administrator",
	"administrators",
	"admins",
	"atname",
	"example",
	"me",
	"self",
}

// IsValidAtnameは、アットネームがすべてのアカウントに課される形式・長さ・予約語の規則を
// 満たすかを返す。サインアップフォームの外でアカウントを作る呼び出し元が、制約の写しでは
// なく同じ規則を参照できるよう公開している。
//
// フォームのバリデーター (addAtnameFormatErrors) は同じ規則を個別に適用する。フォームは規則
// ごとに別のメッセージで伝え分けるため。ここへ規則を足したときは、あちらへも足す必要がある。
// そうしないと、フォームが受理するatnameを名簿が拒否することになる。
func IsValidAtname(atname string) bool {
	return len(atname) <= AtnameMaxLength && atnameRegex.MatchString(atname) && !isReservedAtname(atname)
}

func isReservedAtname(atname string) bool {
	return slices.Contains(reservedAtnames, atname)
}

// addAtnameFormatErrorsは、アットネームの入力・長さ・形式・予約語の規則に反していれば
// フィールドのエラーを積む。アカウントの作成とプロフィールの編集が同じ規則を使う。
func addAtnameFormatErrors(ctx context.Context, ve *model.ValidationError, atname string) {
	if atname == "" {
		ve.AddField("atname", i18n.T(ctx, "validation_atname_required"))
		return
	}
	if len(atname) > AtnameMaxLength {
		ve.AddField("atname", i18n.T(ctx, "validation_atname_too_long"))
	}
	if !atnameRegex.MatchString(atname) {
		ve.AddField("atname", i18n.T(ctx, "validation_atname_invalid_format"))
	}
	if isReservedAtname(atname) {
		ve.AddField("atname", i18n.T(ctx, "validation_atname_reserved"))
	}
}

// validateAtnameAvailabilityは、アットネームをほかのユーザーが使っていれば検証エラーを返す。
// excludeIDを渡すと、そのユーザー自身は数えない (自分の今のアットネームのまま保存できるように
// するため)。アットネームの一意インデックスは削除済みのユーザーも含むため、削除済みのユーザーが
// 使っているアットネームも使えないものとして扱う。
func validateAtnameAvailability(ctx context.Context, userRepo *repository.UserRepository, atname string, excludeID *model.UserID) error {
	exists, err := userRepo.ExistsByAtname(ctx, atname, excludeID)
	if err != nil {
		return err
	}
	if exists {
		ve := model.NewValidationError()
		ve.AddField("atname", i18n.T(ctx, "validation_atname_already_taken"))
		return ve
	}
	return nil
}
