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

// seed-wiki and seed-solo are created so that both answers to "have I joined
// this space?" can be seen from one session. seed-wiki holds everything worth browsing and
// every account has joined it. seed-solo holds only roleOwner, so signing in as
// roleCollaborator or roleGuest shows what a space looks like from the outside
// — as roleGuest with every feature flag on, which is what puts the Go version
// of those screens in front of a non-member.
//
// [Ja] seed-wiki と seed-solo を作るのは、「自分はこのスペースに参加しているか」の
// 両方の答えを 1 つのセッションから確認できるようにするため。seed-wiki は閲覧対象を
// 集めたスペースで、すべてのアカウントが参加している。seed-solo に参加しているのは
// roleOwner だけであるため、roleCollaborator か roleGuest でサインインするとスペースを
// 外側から見た状態を確認できる。roleGuest はフィーチャーフラグを全件持つため、
// そこで見えるのは画面の Go 版になる。
const (
	wikiSpaceIdentifier = "seed-wiki"
	wikiSpaceName       = "シード Wiki"
	soloSpaceIdentifier = "seed-solo"
	soloSpaceName       = "シード個人スペース"
)

// seed-long-name carries a name as long as a space name may be. A space name is
// drawn where there is little room for it — the sidebar, and the header above
// every screen inside the space — so keeping one name that cannot be shortened
// is how those places are checked for whether they wrap it or let it overflow.
// Every other space in the seed is named briefly, which leaves this one as the
// only place the question is asked.
//
// The name is 30 characters, the longest Space::NAME_MAX_LENGTH allows on the
// Rails side, which still owns space creation.
//
// Only roleOwner joins it. What the space is here to show is how a name is
// drawn, not how membership changes what is seen; seed-wiki and seed-solo
// already cover that.
//
// [Ja] seed-long-name は、スペース名が取りうる最大の長さの名前を持つ。スペース名は
// 幅の余裕が無い場所へ描かれる。サイドバーや、スペース内の各画面の上に出る
// ヘッダーである。短くできない名前を 1 つ置いておくことが、それらの場所が名前を
// 折り返すのか、はみ出させるのかを確認する方法になる。シードの他のスペースはどれも
// 短い名前を持つため、この問いを投げかける場所はここだけになる。
//
// 名前は 30 文字で、スペースの作成を今も担当している Rails 側の
// Space::NAME_MAX_LENGTH が許す最大の長さ。
//
// 参加するのは roleOwner だけ。このスペースが示すのは名前の描かれ方であって、
// メンバーシップによる見え方の違いではない。それは seed-wiki と seed-solo が既に
// 担っている。
const (
	longNameSpaceIdentifier = "seed-long-name"
	longNameSpaceName       = "折り返しの確認用に名前を最大文字数まで伸ばしたシードスペース"
)

// demo is the space the screenshots on the help pages are taken in. Every other
// space in the seed is named for what a developer checks there, and a name of
// that kind reaching a help page would be read as part of the product rather
// than as scaffolding for a picture. This one is named as one person's own
// space, which is what a space is created as today.
//
// The identifier is where that matters most: it is the space's part of every
// page URL, so it is photographed together with the screen. It is the one
// space identifier in the seed without the seed- prefix for that reason.
//
// Only roleOwner joins it, the account the browser verification signs in as.
// What this space shows is a wiki that has been written in, not what
// membership changes about the view; seed-wiki and seed-solo already cover
// that.
//
// [Ja] demo は、ヘルプページに載せるスクリーンショットを撮るためのスペース。
// シードの他のスペースはいずれも、開発者がそこで何を確認するかを名前にしている。
// その種の名前がヘルプページへ届くと、絵のための足場ではなくプロダクトの一部と
// して読まれてしまう。このスペースには、今のところスペースがそう作られるもので
// ある「個人が自分のために作ったスペース」としての名前を与える。
//
// とりわけ効くのは識別子で、これはすべてのページ URL に入るスペース側の部分で
// あるため、画面と一緒に写り込む。シードのスペース識別子の中でこれだけが
// seed- 接頭辞を持たないのはそのためである。
//
// 参加するのは roleOwner だけで、これはブラウザ確認でサインインするアカウント。
// このスペースが見せるのは、実際に書かれた Wiki であって、メンバーシップによる
// 見え方の違いではない。それは seed-wiki と seed-solo が既に担っている。
const (
	demoSpaceIdentifier = "demo"
	demoSpaceName       = "みゆきのスペース"
)

// adminSpaceScopes is what production grants the member who creates a space:
// space:admin, the one scope that expands to every other.
//
// [Ja] adminSpaceScopes は、スペースを作成したメンバーに本番が与えるスコープ。
// space:admin は他のすべてのスコープへ展開される唯一の特別スコープ。
var adminSpaceScopes = []model.Scope{model.ScopeSpaceAdmin}

// nonAdminSpaceScopes is what the accounts that do not administer seed-wiki
// hold there, which are roleCollaborator and roleGuest: enough to write pages,
// drafts, suggestions and comments, but no space:admin. Keeping accounts on
// each side of that line is what turns the admin-only parts of a screen into a
// visible difference rather than something that has to be taken on trust.
//
// The two hold the same set so that the feature flags are the only thing that
// separates them. A screen compared between them then shows what the flag
// changes, rather than what the flag and a different set of scopes change
// together, and the Go version of the non-admin screens becomes reachable at
// all: roleOwner, the other account holding the flags, sees every screen as an
// administrator.
//
// topic:read is deliberately absent. Scopes on the space membership apply to
// every topic in the space, so granting it here would reveal the private topics
// these accounts have not joined and erase the only case those topics exist to
// show. The private topic roleCollaborator has joined is opened by the scope on
// that topic membership instead (see topicMemberScopes).
//
// [Ja] nonAdminSpaceScopes は、seed-wiki を管理しないアカウント、すなわち
// roleCollaborator と roleGuest がそこで持つスコープ。ページ・下書き・編集提案・
// コメントを書けるだけの権限を与え、space:admin は持たせない。この線の両側に
// アカウントを置くことで、画面の管理者専用部分が、信じるしかないものではなく
// 目に見える差分になる。
//
// 2 つに同じ集合を持たせるのは、両者を隔てるものをフィーチャーフラグだけにする
// ため。そうすると、2 つの間で見比べた画面が示すものは、フラグと異なるスコープが
// 一緒に変えたものではなく、フラグが変えたものになる。加えて、管理者以外の画面の
// Go 版がそもそも開けるようになる。フラグを持つもう一方のアカウントである
// roleOwner は、すべての画面を管理者として見るためである。
//
// topic:read は意図的に外している。スペースメンバーのスコープはスペース内の全
// トピックに効くため、ここで与えるとこれらのアカウントが参加していない非公開
// トピックまで見えてしまい、そのトピックを置いた唯一の目的が失われる。
// roleCollaborator が参加している非公開トピックは、そのトピックメンバー側の
// スコープで開く
// (topicMemberScopes 参照)。
var nonAdminSpaceScopes = []model.Scope{
	model.ScopePageWrite,
	model.ScopePageTrash,
	model.ScopePageRestore,
	model.ScopeDraftPageWrite,
	model.ScopeDraftPageDelete,
	model.ScopeSuggestionWrite,
	model.ScopeSuggestionCommentWrite,
	model.ScopeAttachmentWrite,
}

// seededSpaceMember is one account's membership in a space. The scopes travel
// with it because what a topic membership has to carry depends on what the
// space membership already grants.
//
// The display name travels with it for the same kind of reason: the text a
// generator writes into a space names the accounts of that space, and the
// membership is the handle it already holds for them. Carrying the name here is
// what lets that text follow the roster instead of repeating what the roster
// said on the day the text was written.
//
// [Ja] seededSpaceMember は 1 アカウントのスペースへのメンバーシップ。スコープを
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

// seededSpace is one created space together with the memberships inside it.
// The generators that follow write their rows as one of these members, and the
// space_id that every one of their queries carries comes from here.
//
// [Ja] seededSpace は作成したスペース 1 つと、その中のメンバーシップ。後続の
// 生成器はこのメンバーのいずれかとして行を書き、それらのクエリが必ず持つ
// space_id もここから取る。
type seededSpace struct {
	id model.SpaceID
	// identifier is the space's part of every page URL. It travels with the
	// space because rendering a body turns wiki links into hrefs, and those need
	// it.
	//
	// [Ja] identifier は、すべてのページ URL に入るスペース側の部分。本文の
	// レンダリングが Wiki リンクを href に変換する際に必要になるため、スペースと
	// 一緒に持たせている。
	identifier model.SpaceIdentifier
	// members are the memberships inside the space, keyed by the role of the
	// account each one belongs to. A role that has not joined is absent rather
	// than present without scopes: seed-solo is looked at from an account that
	// is not a member of it at all.
	//
	// [Ja] members はスペース内のメンバーシップを、それぞれが属するアカウントの
	// 役割をキーにして持つ。参加していない役割は、スコープ無しで存在するのではなく
	// 存在しない。seed-solo は、そのメンバーではまったくないアカウントから眺める
	// ものであるため。
	members map[seedRole]*seededSpaceMember
}

// member returns the membership the account of the role holds in the space, or
// nil when that account has not joined it.
//
// [Ja] member は、その役割のアカウントがスペース内で持つメンバーシップを返す。
// 参加していない場合は nil を返す。
func (s *seededSpace) member(role seedRole) *seededSpaceMember {
	return s.members[role]
}

// requireMember returns the membership the account of the role holds in the
// space, and an error naming the role when that account has not joined it. A
// generator that cannot do its work without a role asks for it this way, so
// that a role missing from the space is reported where it is asked for instead
// of reaching a write as a nil membership.
//
// [Ja] requireMember は、その役割のアカウントがスペース内で持つメンバーシップを
// 返す。参加していない場合は、その役割を名指しするエラーを返す。ある役割が無いと
// 仕事にならない生成器はこちらで求める。スペースに参加していない役割が、nil の
// メンバーシップのまま書き込みへ届くのではなく、求めた場所で報告されるようにする
// ため。
func (s *seededSpace) requireMember(role seedRole) (*seededSpaceMember, error) {
	member := s.member(role)
	if member == nil {
		return nil, fmt.Errorf("スペース %s に役割 %s が参加していない", s.identifier, role)
	}

	return member, nil
}

// memberInTurn returns the membership that takes the item at the given 1-based
// position, handing the items round the roles in the order they are listed.
//
// The roles are named by the caller rather than read off the space, so neither
// an empty list nor a role that has not joined can be ruled out here by
// construction. Both are returned as an error: left alone, the first divides by
// zero and the second travels on as a nil membership.
//
// [Ja] memberInTurn は、1 始まりの位置にある項目を担当するメンバーシップを返す。
// 項目は、並べられた順に役割へ回される。
//
// 役割の一覧はスペースから読むのではなく呼び出し側が名指しするため、空の一覧も、
// 参加していない役割も、この関数の構造だけでは排除できない。どちらもエラーとして
// 返す。放置すると、前者はゼロ除算になり、後者は nil のメンバーシップのまま先へ
// 運ばれるため。
func (s *seededSpace) memberInTurn(roles []seedRole, position int) (*seededSpaceMember, error) {
	if len(roles) == 0 {
		return nil, fmt.Errorf("スペース %s の担当に指定された役割が 1 つも無い", s.identifier)
	}

	return s.requireMember(roles[(position-1)%len(roles)])
}

// spaceMemberSpec is one membership to create in a space.
//
// [Ja] spaceMemberSpec は、スペース内に作成するメンバーシップ 1 件の内容。
type spaceMemberSpec struct {
	role   seedRole
	scopes []model.Scope
}

// spaceSpec describes one space to create.
//
// [Ja] spaceSpec は、作成するスペース 1 件の内容。
type spaceSpec struct {
	identifier string
	name       string
	members    []spaceMemberSpec
	// assign stores the created space on the result.
	//
	// [Ja] assign は、作成したスペースを結果へ格納する。
	assign func(spaces *seededSpaces, space *seededSpace)
}

// spaceSpecs are the spaces to create, together with who joins each one and
// with what.
//
// [Ja] spaceSpecs は作成するスペースと、それぞれに誰が何を持って参加するか。
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

// seededSpaces holds the spaces the seed created.
//
// [Ja] seededSpaces はシードが作成したスペースを保持する。
type seededSpaces struct {
	wiki *seededSpace
	solo *seededSpace
	// longName is the space that carries the longest names the model allows,
	// both in its own name and in the names of its topics.
	//
	// [Ja] longName は、モデルが許す最長の名前を、自身の名前とトピックの名前の
	// 双方で持つスペース。
	longName *seededSpace
	// demo is the space the help pages are photographed in. It stands apart from
	// the three above because those are arranged to cover the states a screen can
	// be in, while this one is arranged to read as a wiki somebody keeps. Putting
	// either kind of page into the other space would cost both of them what they
	// are for.
	//
	// [Ja] demo は、ヘルプページのスクリーンショットを撮るスペース。上の 3 つと
	// 別に置いているのは、あちらが画面の取りうる状態を網羅するために並べられて
	// いるのに対し、こちらは誰かが持っている Wiki として読まれるために並べられて
	// いるため。どちらのページをもう一方のスペースへ入れても、双方が何のために
	// あるのかを損なう。
	demo *seededSpace
}

// generateSpaces creates the spaces and the memberships that decide what each
// account may do in them.
//
// [Ja] generateSpaces はスペースと、各アカウントがそこで何をできるかを決める
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
				return nil, fmt.Errorf("スペース %s に参加させる役割 %s のユーザーが作成されていない", spec.identifier, memberSpec.role)
			}

			member, err := addSpaceMember(ctx, dbtx, space.id, user, memberSpec.scopes)
			if err != nil {
				return nil, fmt.Errorf("スペース %s への役割 %s の追加に失敗: %w", spec.identifier, memberSpec.role, err)
			}
			space.members[memberSpec.role] = member
		}

		spec.assign(spaces, space)
		bar.advance()
	}

	return spaces, nil
}

// createSpace inserts one space.
//
// The row is written here rather than through a repository because creating a
// space is still handled by the Rails side, and the Go side has no Create to
// call.
//
// [Ja] createSpace はスペースを 1 つ INSERT する。
//
// Repository ではなくここで行を書くのは、スペースの作成を担当しているのが今も
// Rails 側であり、Go 側に呼べる Create が無いため。
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
		return nil, fmt.Errorf("スペース %s の作成に失敗: %w", identifier, err)
	}

	return &seededSpace{
		id:         model.SpaceID(id),
		identifier: model.SpaceIdentifier(identifier),
		members:    make(map[seedRole]*seededSpaceMember),
	}, nil
}

// addSpaceMember joins a user to a space with the given scopes.
//
// [Ja] addSpaceMember は、指定のスコープでユーザーをスペースに参加させる。
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

// scopeStrings converts scopes for storage. The result is never nil: the
// scopes column is NOT NULL, and pq sends a nil slice as NULL rather than as
// the empty array a membership without scopes needs.
//
// [Ja] scopeStrings はスコープを保存用に変換する。結果が nil になることはない。
// scopes 列は NOT NULL であり、pq は nil のスライスを、スコープ無しの
// メンバーシップに必要な空配列ではなく NULL として送るため。
func scopeStrings(scopes []model.Scope) []string {
	ss := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		ss = append(ss, string(scope))
	}

	return ss
}
