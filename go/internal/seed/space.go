package seed

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// seed-wikiとseed-soloを作るのは、「自分はこのスペースに参加しているか」の
// 両方の答えを1つのセッションから確認できるようにするため。seed-wikiは閲覧対象を
// 集めたスペースで、すべてのアカウントが参加している。seed-soloに参加しているのは
// roleOwnerだけであるため、roleCollaboratorかroleGuestでサインインするとスペースを
// 外側から見た状態を確認できる。roleGuestはフィーチャーフラグを全件持つため、
// そこで見えるのは画面のGo版になる。
const (
	wikiSpaceIdentifier = "seed-wiki"
	wikiSpaceName       = "シード Wiki"
	soloSpaceIdentifier = "seed-solo"
	soloSpaceName       = "シード個人スペース"
)

// seed-long-nameは、スペース名が取りうる最大の長さの名前を持つ。スペース名は
// 幅の余裕が無い場所へ描かれる。サイドバーや、スペース内の各画面の上に出る
// ヘッダーである。短くできない名前を1つ置いておくことが、それらの場所が名前を
// 折り返すのか、はみ出させるのかを確認する方法になる。シードの他のスペースはどれも
// 短い名前を持つため、この問いを投げかける場所はここだけになる。
//
// 名前は30文字で、スペースの作成を今も担当しているRails側の
// Space::NAME_MAX_LENGTHが許す最大の長さ。
//
// 参加するのはroleOwnerだけ。このスペースが示すのは名前の描かれ方であって、
// メンバーシップによる見え方の違いではない。それはseed-wikiとseed-soloが既に
// 担っている。
const (
	longNameSpaceIdentifier = "seed-long-name"
	longNameSpaceName       = "折り返しの確認用に名前を最大文字数まで伸ばしたシードスペース"
)

// demoは、ヘルプページに載せるスクリーンショットを撮るためのスペース。
// シードの他のスペースはいずれも、開発者がそこで何を確認するかを名前にしている。
// その種の名前がヘルプページへ届くと、絵のための足場ではなくプロダクトの一部と
// して読まれてしまう。このスペースには、今のところスペースがそう作られるもので
// ある「個人が自分のために作ったスペース」としての名前を与える。
//
// とりわけ効くのは識別子で、これはすべてのページURLに入るスペース側の部分で
// あるため、画面と一緒に写り込む。シードのスペース識別子の中でこれだけが
// seed- 接頭辞を持たないのはそのためである。
//
// 参加するのはroleOwnerだけで、これはブラウザ確認でサインインするアカウント。
// このスペースが見せるのは、実際に書かれたWikiであって、メンバーシップによる
// 見え方の違いではない。それはseed-wikiとseed-soloが既に担っている。
const (
	demoSpaceIdentifier = "demo"
	demoSpaceName       = "みゆきのスペース"
)

// adminSpaceScopesは、スペースを作成したメンバーに本番が与えるスコープ。
// space:adminは他のすべてのスコープへ展開される唯一の特別スコープ。
var adminSpaceScopes = []model.Scope{model.ScopeSpaceAdmin}

// nonAdminSpaceScopesは、seed-wikiを管理しないアカウント、すなわち
// roleCollaboratorとroleGuestがそこで持つスコープ。ページ・下書き・編集提案・
// コメントを書けるだけの権限を与え、space:adminは持たせない。この線の両側に
// アカウントを置くことで、画面の管理者専用部分が、信じるしかないものではなく
// 目に見える差分になる。
//
// 2つに同じ集合を持たせるのは、両者を隔てるものをフィーチャーフラグだけにする
// ため。そうすると、2つの間で見比べた画面が示すものは、フラグと異なるスコープが
// 一緒に変えたものではなく、フラグが変えたものになる。加えて、管理者以外の画面の
// Go版がそもそも開けるようになる。フラグを持つもう一方のアカウントである
// roleOwnerは、すべての画面を管理者として見るためである。
//
// topic:readは意図的に外している。スペースメンバーのスコープはスペース内の全
// トピックに効くため、ここで与えるとこれらのアカウントが参加していない非公開
// トピックまで見えてしまい、そのトピックを置いた唯一の目的が失われる。
// roleCollaboratorが参加している非公開トピックは、そのトピックメンバー側の
// スコープで開く
// (topicMemberScopes参照)。
var nonAdminSpaceScopes = []model.Scope{
	model.ScopePageWrite,
	model.ScopePageTrashWrite,
	model.ScopePageTrashDelete,
	model.ScopeDraftPageWrite,
	model.ScopeDraftPageDelete,
	model.ScopeSuggestionWrite,
	model.ScopeSuggestionCommentWrite,
	model.ScopeAttachmentWrite,
}

// seededSpaceMemberは1アカウントのスペースへのメンバーシップ。スコープを
// 一緒に持たせるのは、トピックメンバーシップが何を持つべきかが、スペース
// メンバーシップが既に与えているものによって決まるため。
//
// 表示名を一緒に持たせるのも同じ種類の理由による。生成器がスペースへ書き込む
// テキストが名指しするのはそのスペースのアカウントであり、メンバーシップは生成器が
// そのために既に持っている手がかりである。名前をここに持たせていることが、その
// テキストが、書かれた日に名簿が言っていたことを繰り返すのではなく、名簿へ追随
// できる理由になる。
type seededSpaceMember struct {
	id     model.SpaceMemberID
	name   string
	scopes []model.Scope
}

// seededSpaceは作成したスペース1つと、その中のメンバーシップ。後続の
// 生成器はこのメンバーのいずれかとして行を書き、それらのクエリが必ず持つ
// space_idもここから取る。
type seededSpace struct {
	id model.SpaceID
	// identifierは、すべてのページURLに入るスペース側の部分。本文の
	// レンダリングがWikiリンクをhrefに変換する際に必要になるため、スペースと
	// 一緒に持たせている。
	identifier model.SpaceIdentifier
	// membersはスペース内のメンバーシップを、それぞれが属するアカウントの
	// 役割をキーにして持つ。参加していない役割は、スコープ無しで存在するのではなく
	// 存在しない。seed-soloは、そのメンバーではまったくないアカウントから眺める
	// ものであるため。
	members map[seedRole]*seededSpaceMember
}

// memberは、その役割のアカウントがスペース内で持つメンバーシップを返す。
// 参加していない場合はnilを返す。
func (s *seededSpace) member(role seedRole) *seededSpaceMember {
	return s.members[role]
}

// requireMemberは、その役割のアカウントがスペース内で持つメンバーシップを
// 返す。参加していない場合は、その役割を名指しするエラーを返す。ある役割が無いと
// 仕事にならない生成器はこちらで求める。スペースに参加していない役割が、nilの
// メンバーシップのまま書き込みへ届くのではなく、求めた場所で報告されるようにする
// ため。
func (s *seededSpace) requireMember(role seedRole) (*seededSpaceMember, error) {
	member := s.member(role)
	if member == nil {
		return nil, fmt.Errorf("スペース %sに役割 %sが参加していない", s.identifier, role)
	}

	return member, nil
}

// memberInTurnは、1始まりの位置にある項目を担当するメンバーシップを返す。
// 項目は、並べられた順に役割へ回される。
//
// 役割の一覧はスペースから読むのではなく呼び出し側が名指しするため、空の一覧も、
// 参加していない役割も、この関数の構造だけでは排除できない。どちらもエラーとして
// 返す。放置すると、前者はゼロ除算になり、後者はnilのメンバーシップのまま先へ
// 運ばれるため。
func (s *seededSpace) memberInTurn(roles []seedRole, position int) (*seededSpaceMember, error) {
	if len(roles) == 0 {
		return nil, fmt.Errorf("スペース %sの担当に指定された役割が1つも無い", s.identifier)
	}

	return s.requireMember(roles[(position-1)%len(roles)])
}

// spaceMemberSpecは、スペース内に作成するメンバーシップ1件の内容。
type spaceMemberSpec struct {
	role   seedRole
	scopes []model.Scope
}

// spaceSpecは、作成するスペース1件の内容。
type spaceSpec struct {
	identifier string
	name       string
	members    []spaceMemberSpec
	// assignは、作成したスペースを結果へ格納する。
	assign func(spaces *seededSpaces, space *seededSpace)
}

// spaceSpecsは作成するスペースと、それぞれに誰が何を持って参加するか。
var spaceSpecs = []spaceSpec{
	{
		identifier: wikiSpaceIdentifier,
		name:       wikiSpaceName,
		members: []spaceMemberSpec{
			{role: roleOwner, scopes: adminSpaceScopes},
			{role: roleCollaborator, scopes: nonAdminSpaceScopes},
			{role: roleGuest, scopes: nonAdminSpaceScopes},
		},
		assign: func(spaces *seededSpaces, space *seededSpace) { spaces.wiki = space },
	},
	{
		identifier: soloSpaceIdentifier,
		name:       soloSpaceName,
		members: []spaceMemberSpec{
			{role: roleOwner, scopes: adminSpaceScopes},
		},
		assign: func(spaces *seededSpaces, space *seededSpace) { spaces.solo = space },
	},
	{
		identifier: longNameSpaceIdentifier,
		name:       longNameSpaceName,
		members: []spaceMemberSpec{
			{role: roleOwner, scopes: adminSpaceScopes},
		},
		assign: func(spaces *seededSpaces, space *seededSpace) { spaces.longName = space },
	},
	{
		identifier: demoSpaceIdentifier,
		name:       demoSpaceName,
		members: []spaceMemberSpec{
			{role: roleOwner, scopes: adminSpaceScopes},
		},
		assign: func(spaces *seededSpaces, space *seededSpace) { spaces.demo = space },
	},
}

// seededSpacesはシードが作成したスペースを保持する。
type seededSpaces struct {
	wiki *seededSpace
	solo *seededSpace
	// longNameは、モデルが許す最長の名前を、自身の名前とトピックの名前の
	// 双方で持つスペース。
	longName *seededSpace
	// demoは、ヘルプページのスクリーンショットを撮るスペース。上の3つと
	// 別に置いているのは、あちらが画面の取りうる状態を網羅するために並べられて
	// いるのに対し、こちらは誰かが持っているWikiとして読まれるために並べられて
	// いるため。どちらのページをもう一方のスペースへ入れても、双方が何のために
	// あるのかを損なう。
	demo *seededSpace
}

// generateSpacesはスペースと、各アカウントがそこで何をできるかを決める
// メンバーシップを作成する。
func generateSpaces(ctx context.Context, dbtx query.DBTX, out io.Writer, users *seededUsers) (*seededSpaces, error) {
	bar := newProgress(out, "スペース", len(spaceSpecs))
	defer bar.finish()

	spaces := &seededSpaces{}
	for _, spec := range spaceSpecs {
		space, err := createSpace(ctx, dbtx, spec.identifier, spec.name)
		if err != nil {
			return nil, err
		}

		for _, memberSpec := range spec.members {
			user := users.user(memberSpec.role)
			if user == nil {
				return nil, fmt.Errorf("スペース %sに参加させる役割 %sのユーザーが作成されていない", spec.identifier, memberSpec.role)
			}

			member, err := addSpaceMember(ctx, dbtx, space.id, user, memberSpec.scopes)
			if err != nil {
				return nil, fmt.Errorf("スペース %sへの役割 %sの追加に失敗: %w", spec.identifier, memberSpec.role, err)
			}
			space.members[memberSpec.role] = member
		}

		spec.assign(spaces, space)
		bar.advance()
	}

	return spaces, nil
}

// createSpaceはスペースを1つINSERTする。
//
// Repositoryではなくここで行を書くのは、スペースの作成を担当しているのが今も
// Rails側であり、Go側に呼べるCreateが無いため。
func createSpace(ctx context.Context, dbtx query.DBTX, identifier string, name string) (*seededSpace, error) {
	now := time.Now()

	var id string
	err := dbtx.QueryRowContext(
		ctx,
		`INSERT INTO spaces (identifier, name, plan, joined_at, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6)
         RETURNING id`,
		identifier, name, int32(model.PlanFree), now, now, now,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("スペース %sの作成に失敗: %w", identifier, err)
	}

	return &seededSpace{
		id:         model.SpaceID(id),
		identifier: model.SpaceIdentifier(identifier),
		members:    make(map[seedRole]*seededSpaceMember),
	}, nil
}

// addSpaceMemberは、指定のスコープでユーザーをスペースに参加させる。
func addSpaceMember(
	ctx context.Context,
	dbtx query.DBTX,
	spaceID model.SpaceID,
	user *model.User,
	scopes []model.Scope,
) (*seededSpaceMember, error) {
	now := time.Now()

	var id string
	err := dbtx.QueryRowContext(
		ctx,
		`INSERT INTO space_members (space_id, user_id, scopes, joined_at, active, created_at, updated_at)
         VALUES ($1, $2, $3, $4, true, $5, $6)
         RETURNING id`,
		string(spaceID), string(user.ID), pq.Array(scopeStrings(scopes)), now, now, now,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	return &seededSpaceMember{id: model.SpaceMemberID(id), name: user.Name, scopes: scopes}, nil
}

// scopeStringsはスコープを保存用に変換する。結果がnilになることはない。
// scopes列はNOT NULLであり、pqはnilのスライスを、スコープ無しの
// メンバーシップに必要な空配列ではなくNULLとして送るため。
func scopeStrings(scopes []model.Scope) []string {
	ss := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		ss = append(ss, string(scope))
	}

	return ss
}
