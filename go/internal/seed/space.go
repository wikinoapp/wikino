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

// Two spaces are created so that both answers to "have I joined this space?"
// can be seen from one session. seed-wiki holds everything worth browsing and
// every account has joined it. seed-solo holds only roleOwner, so signing in as
// roleCollaborator or roleGuest shows what a space looks like from the outside
// — as roleGuest with every feature flag on, which is what puts the Go version
// of those screens in front of a non-member.
//
// [Ja] スペースを 2 つ作るのは、「自分はこのスペースに参加しているか」の両方の
// 答えを 1 つのセッションから確認できるようにするため。seed-wiki は閲覧対象を
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
}

// seededSpaces holds both spaces the seed created.
//
// [Ja] seededSpaces はシードが作成した 2 つのスペースを保持する。
type seededSpaces struct {
	wiki *seededSpace
	solo *seededSpace
}

// generateSpaces creates the two spaces and the memberships that decide what
// each account may do in them.
//
// [Ja] generateSpaces は 2 つのスペースと、各アカウントがそこで何をできるかを
// 決めるメンバーシップを作成する。
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
