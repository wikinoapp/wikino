package seed

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestGenerateSpaces(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	users := buildSeedUsers(t, tx, "seed-spaces")

	spaces, err := generateSpaces(ctx, tx, io.Discard, users)
	if err != nil {
		t.Fatalf("スペース生成に失敗: %v", err)
	}

	assertSpace(ctx, t, tx, spaces.wiki, wikiSpaceIdentifier, "シード Wiki")
	assertSpace(ctx, t, tx, spaces.solo, soloSpaceIdentifier, "シード個人スペース")
	assertSpace(ctx, t, tx, spaces.longName, longNameSpaceIdentifier, "折り返しの確認用に名前を最大文字数まで伸ばしたシードスペース")
	assertSpace(ctx, t, tx, spaces.demo, demoSpaceIdentifier, "みゆきのスペース")

	// seed-solo exists to be opened by an account that has not joined it, so
	// neither of the two roles that open it that way may have a membership row
	// there at all. A membership with no scopes would not do: the space would
	// still be listed as one of that account's own.
	//
	// roleGuest is the one that matters most here. It holds every feature flag,
	// so a membership written for it by mistake would not fail anything — it
	// would quietly turn the one account that reaches these screens as a
	// non-member into another member of the space.
	//
	// [Ja] seed-solo は、参加していないアカウントから開くために存在するため、
	// そのように開く 2 つの役割のどちらについても、メンバーシップ行がそもそも
	// 無いことが必要になる。スコープ無しのメンバーシップでは足りない。それでも
	// スペースはそのアカウント自身のものとして一覧に出てしまうため。
	//
	// ここで特に効くのは roleGuest のほう。フィーチャーフラグを全件持つため、
	// 誤ってメンバーシップが書かれても何も失敗せず、これらの画面へ非メンバーとして
	// 辿り着く唯一のアカウントが、黙ってスペースのメンバーに変わるだけになる。
	for _, role := range []seedRole{roleCollaborator, roleGuest} {
		if spaces.solo.member(role) != nil {
			t.Errorf("seed-soloには%sが参加していないことを期待したがメンバーが返された", role)
		}
	}
	assertSpaceMemberCount(ctx, t, tx, spaces.wiki.id, 3)
	assertSpaceMemberCount(ctx, t, tx, spaces.solo.id, 1)
	assertSpaceMemberCount(ctx, t, tx, spaces.longName.id, 1)

	// The demo space is photographed for the help pages, so a second account
	// joining it would put a name nobody outside the seed knows into the member
	// list of a picture. Counting the rows is what says roleOwner is alone there;
	// asking the result for the other roles would only report what the spec named.
	//
	// [Ja] デモスペースはヘルプページのために撮影されるため、2 つ目のアカウントが
	// 参加すると、シードの外の誰も知らない名前が画像のメンバー一覧に載ってしまう。
	// roleOwner だけであることを述べるのは行数のほうである。他の役割を結果へ
	// 尋ねても、仕様が名指しした内容が返るだけになる。
	assertSpaceMemberCount(ctx, t, tx, spaces.demo.id, 1)

	for _, tt := range []struct {
		label   string
		role    seedRole
		spaceID model.SpaceID
		member  *seededSpaceMember
		want    []model.Scope
	}{
		{label: "seed-wikiのowner", role: roleOwner, spaceID: spaces.wiki.id, member: spaces.wiki.member(roleOwner), want: adminSpaceScopes},
		{label: "seed-wikiのcollaborator", role: roleCollaborator, spaceID: spaces.wiki.id, member: spaces.wiki.member(roleCollaborator), want: nonAdminSpaceScopes},
		{label: "seed-wikiのguest", role: roleGuest, spaceID: spaces.wiki.id, member: spaces.wiki.member(roleGuest), want: nonAdminSpaceScopes},
		{label: "seed-soloのowner", role: roleOwner, spaceID: spaces.solo.id, member: spaces.solo.member(roleOwner), want: adminSpaceScopes},
		{label: "demoのowner", role: roleOwner, spaceID: spaces.demo.id, member: spaces.demo.member(roleOwner), want: adminSpaceScopes},
	} {
		assertSpaceMemberScopes(ctx, t, tx, tt.label, tt.spaceID, tt.member.id, tt.want)
		if got, want := tt.member.name, users.user(tt.role).Name; got != want {
			t.Errorf("%sの表示名が %q であることを期待したが %q だった", tt.label, want, got)
		}
	}
}

func TestGenerateSpacesRejectsRoleWithoutUser(t *testing.T) {
	t.Parallel()

	for _, missingRole := range []seedRole{roleOwner, roleCollaborator, roleGuest} {
		t.Run(string(missingRole), func(t *testing.T) {
			_, tx := testutil.SetupTx(t)
			ctx := context.Background()

			users := buildSeedUsers(t, tx, "seed-spaces-no-"+string(missingRole))
			delete(users.byRole, missingRole)

			_, err := generateSpaces(ctx, tx, io.Discard, users)
			wantErr := "スペース seed-wiki に参加させる役割 " + string(missingRole) + " のユーザーが作成されていない"
			if err == nil {
				t.Fatal("ユーザー不足のエラーを期待したがnilだった")
			}
			if err.Error() != wantErr {
				t.Errorf("エラーが %q であることを期待したが %q だった", wantErr, err)
			}
		})
	}
}

func TestSeededSpaceRequireMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	spaces := buildSeedSpaces(t, tx, "seed-require-member")

	member, err := spaces.wiki.requireMember(roleCollaborator)
	if err != nil {
		t.Fatalf("参加している役割で予期しないエラー: %v", err)
	}
	if member != spaces.wiki.member(roleCollaborator) {
		t.Error("requireMemberがmemberと異なるメンバーシップを返した")
	}

	// A role that has not joined has to be named where it is asked for. Left as
	// a nil membership, it would only fail once it reached a write, with nothing
	// in the message to say which role was missing.
	//
	// [Ja] 参加していない役割は、求めた場所で名指しされる必要がある。nil の
	// メンバーシップのままだと、書き込みへ届いて初めて失敗し、どの役割が欠けて
	// いたのかはメッセージに残らない。
	_, err = spaces.solo.requireMember(roleCollaborator)
	wantErr := "スペース seed-require-member-solo に役割 collaborator が参加していない"
	if err == nil {
		t.Fatal("参加していない役割でエラーを期待したがnilだった")
	}
	if err.Error() != wantErr {
		t.Errorf("エラーが %q であることを期待したが %q だった", wantErr, err)
	}
}

func TestSeededSpaceMemberInTurn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	spaces := buildSeedSpaces(t, tx, "seed-member-in-turn")

	// The position is 1-based and wraps, so the roles take the items in the
	// order they are listed. This is what decides who each page, suggestion and
	// remark is attributed to, so the mapping is fixed here rather than left to
	// the generators that read it.
	//
	// [Ja] 位置は 1 始まりで、一覧の長さで巡回する。役割は並べられた順に項目を
	// 担当する。ページ・編集提案・発言を誰のものとして記録するかはこれが決めるため、
	// 対応はそれを読む生成器の側ではなくここで固定する。
	for position, wantRole := range map[int]seedRole{
		1: roleOwner,
		2: roleCollaborator,
		3: roleOwner,
		4: roleCollaborator,
	} {
		member, err := spaces.wiki.memberInTurn(contentAuthorRoles, position)
		if err != nil {
			t.Fatalf("%d 番目の担当の取得に失敗: %v", position, err)
		}
		if member != spaces.wiki.member(wantRole) {
			t.Errorf("%d 番目の担当が役割 %s のメンバーであることを期待したが異なっていた", position, wantRole)
		}
	}

	// An empty list has no role to hand the item to, and would divide by zero if
	// it were indexed.
	//
	// [Ja] 空の一覧には項目を渡す役割が無く、そのまま索引すればゼロ除算になる。
	_, err := spaces.wiki.memberInTurn(nil, 1)
	wantErr := "スペース seed-member-in-turn-wiki の担当に指定された役割が 1 つも無い"
	if err == nil {
		t.Fatal("役割の一覧が空の場合にエラーを期待したがnilだった")
	}
	if err.Error() != wantErr {
		t.Errorf("エラーが %q であることを期待したが %q だった", wantErr, err)
	}

	_, err = spaces.solo.memberInTurn(contentAuthorRoles, 2)
	wantErr = "スペース seed-member-in-turn-solo に役割 collaborator が参加していない"
	if err == nil {
		t.Fatal("参加していない役割が回ってきた場合にエラーを期待したがnilだった")
	}
	if err.Error() != wantErr {
		t.Errorf("エラーが %q であることを期待したが %q だった", wantErr, err)
	}
}

func TestNonAdminSpaceScopes(t *testing.T) {
	t.Parallel()

	publicTopic := &model.Topic{Visibility: model.TopicVisibilityPublic}
	privateTopic := &model.Topic{Visibility: model.TopicVisibilityPrivate}

	// The two scope sets are only worth having if they land on different sides
	// of the checks the screens make. These are the differences the seed data
	// is arranged to produce.
	//
	// [Ja] 2 つのスコープ集合に意味があるのは、画面が行う判定の異なる側に着地する
	// 場合だけ。ここに並ぶのが、シードデータが作り出そうとしている差分そのもの。
	admin := policy.NewMemberPolicy(adminSpaceScopes, nil)
	nonAdmin := policy.NewMemberPolicy(nonAdminSpaceScopes, nil)

	tests := []struct {
		name         string
		check        func(p *policy.MemberPolicy) bool
		wantAdmin    bool
		wantNonAdmin bool
	}{
		{
			name:         "公開トピックはどちらからも見える",
			check:        func(p *policy.MemberPolicy) bool { return p.CanShowTopic(publicTopic) },
			wantAdmin:    true,
			wantNonAdmin: true,
		},
		{
			name:         "参加していない非公開トピックは管理者にしか見えない",
			check:        func(p *policy.MemberPolicy) bool { return p.CanShowTopic(privateTopic) },
			wantAdmin:    true,
			wantNonAdmin: false,
		},
		{
			name:         "ページの作成はどちらもできる",
			check:        func(p *policy.MemberPolicy) bool { return p.CanCreatePage() },
			wantAdmin:    true,
			wantNonAdmin: true,
		},
		{
			name:         "編集提案の作成はどちらもできる",
			check:        func(p *policy.MemberPolicy) bool { return p.CanCreateSuggestion(publicTopic) },
			wantAdmin:    true,
			wantNonAdmin: true,
		},
		{
			name:         "編集提案の適用は管理者にしかできない",
			check:        func(p *policy.MemberPolicy) bool { return p.CanApplySuggestion() },
			wantAdmin:    true,
			wantNonAdmin: false,
		},
		{
			name:         "トピックの作成は管理者にしかできない",
			check:        func(p *policy.MemberPolicy) bool { return p.CanCreateTopic() },
			wantAdmin:    true,
			wantNonAdmin: false,
		},
		{
			name:         "他人の下書きは管理者にしか見えない",
			check:        func(p *policy.MemberPolicy) bool { return p.CanShowDraftPage(false) },
			wantAdmin:    true,
			wantNonAdmin: false,
		},
		{
			name:         "自分の下書きはどちらも見える",
			check:        func(p *policy.MemberPolicy) bool { return p.CanShowDraftPage(true) },
			wantAdmin:    true,
			wantNonAdmin: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.check(admin); got != tt.wantAdmin {
				t.Errorf("管理者 (owner) の判定が %t であることを期待したが %t だった", tt.wantAdmin, got)
			}
			if got := tt.check(nonAdmin); got != tt.wantNonAdmin {
				t.Errorf("非管理者 (collaborator と guest) の判定が %t であることを期待したが %t だった", tt.wantNonAdmin, got)
			}
		})
	}
}

func TestScopeStrings(t *testing.T) {
	t.Parallel()

	// A membership without scopes has to reach the database as an empty array.
	// pq turns a nil slice into NULL, which the NOT NULL column rejects.
	//
	// [Ja] スコープ無しのメンバーシップは、空配列としてデータベースへ届く必要が
	// ある。pq は nil のスライスを NULL に変換し、NOT NULL 列がそれを拒否するため。
	if got := scopeStrings(nil); got == nil || len(got) != 0 {
		t.Errorf("空のスライスを期待したが %#v だった", got)
	}

	got := scopeStrings([]model.Scope{model.ScopeSpaceAdmin, model.ScopeTopicRead})
	want := []string{"space:admin", "topic:read"}
	if len(got) != len(want) {
		t.Fatalf("%d 件を期待したが %d 件だった", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%d 番目が %q であることを期待したが %q だった", i, want[i], got[i])
		}
	}
}

// buildSeedUsers creates the accounts the generators need, without going
// through generateUsers: that function fixes the email and atname of every
// account, and each test that called it would queue up behind the others on the
// unique indexes.
//
// The accounts are keyed by the roles allSeedRoles names, so a role added there
// reaches the generators under test without this helper being edited.
//
// [Ja] buildSeedUsers は生成器が必要とするアカウントを、generateUsers を経由せずに
// 作成する。generateUsers は各アカウントのメールアドレスと atname を固定するため、
// それを呼ぶテストは一意インデックスの上で互いに待ち合わせることになるため。
//
// アカウントは allSeedRoles が挙げる役割をキーにして持つ。そこへ役割を足したとき、
// 本ヘルパーを直さなくてもテスト対象の生成器へ届くようにするため。
func buildSeedUsers(t *testing.T, tx *sql.Tx, prefix string) *seededUsers {
	t.Helper()

	users := &seededUsers{byRole: make(map[seedRole]*model.User, len(allSeedRoles))}
	atnames := make(map[string]seedRole, len(allSeedRoles))
	// The digest is what identifies the prefix, so it is taken once here while
	// the role's initial below is what varies per account.
	//
	// [Ja] prefix を識別するのがダイジェストであるため、ここで 1 度だけ求める。
	// アカウントごとに変わるのは、下で付ける役割の頭文字の方になる。
	digest := sha256.Sum256([]byte(prefix))
	for _, role := range allSeedRoles {
		// Derive a short, deterministic atname from the prefix and keep the
		// role's initial at the end of it. The digest keeps tests with long
		// descriptive prefixes independent on the database's unique indexes
		// without exceeding the production constraint, and the suffix says which
		// role a row belongs to when a failing test is looked into.
		//
		// The initial suffices while no two roles share one. Should they, the
		// check below says so rather than letting the two accounts collide on
		// the unique index.
		//
		// [Ja] prefix から短く決定的な atname を導き、末尾に役割の頭文字を残す。
		// ダイジェストによって、説明的で長い prefix を持つテスト同士をデータベースの
		// 一意インデックス上で独立させながら本番の制約を超えないようにし、接尾辞に
		// よって、失敗したテストを調べるときにその行がどの役割のものかを読み取れる
		// ようにする。
		//
		// 頭文字で足りるのは、頭文字を共有する役割が無い間である。共有するように
		// なったときは、2 つのアカウントを一意インデックスの上で衝突させる代わりに、
		// 下の確認がそう告げる。
		atname := fmt.Sprintf("s%x_%s", digest[:7], role[:1])
		if !validator.IsValidAtname(atname) {
			t.Fatalf("テスト用 atname %q がアプリケーションの制約を満たさない", atname)
		}
		if other, ok := atnames[atname]; ok {
			t.Fatalf("役割 %s と %s のテスト用 atname が %q で衝突している", other, role, atname)
		}
		atnames[atname] = role
		email := atname + "@example.com"
		// The display name says the role in plain sight, because the text the
		// generators write names the accounts it is about. A body or a
		// description that reached for the wrong role would otherwise be checked
		// against a name that says nothing about which account it belongs to.
		//
		// [Ja] 表示名には役割をそのまま出す。生成器が書くテキストは、それが述べて
		// いるアカウントを名指しするため。そうしないと、取り違えた役割へ手を伸ばした
		// 本文や説明文が、どのアカウントのものかを何も語らない名前と突き合わされる
		// ことになる。
		name := "テストユーザー " + string(role)
		id := testutil.NewUserBuilder(t, tx).WithEmail(email).WithAtname(atname).WithName(name).Build()

		users.byRole[role] = &model.User{ID: id, Email: email, Atname: atname, Name: name}
	}

	return users
}

// assertSpace checks that the space carries the expected identifier and name.
// The identifier is checked on the returned space as well as in the database:
// rendering a page body turns wiki links into hrefs from the returned value,
// so the two have to agree. The name is read from the database because it is
// browser-visible data that seededSpace does not need to retain.
//
// [Ja] assertSpace は、スペースが期待する識別子と名前を持つことを確認する。
// 識別子をデータベースだけでなく戻り値でも確認するのは、ページ本文のレンダリングが
// 戻り値から Wiki リンクを href に変換するため。両者が食い違ってはならない。
// 名前はブラウザに表示されるデータだが、seededSpace が保持する必要は無いため、
// データベースから読み取る。
func assertSpace(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	space *seededSpace,
	wantIdentifier string,
	wantName string,
) {
	t.Helper()

	if string(space.identifier) != wantIdentifier {
		t.Errorf("返されたスペース識別子が %q であることを期待したが %q だった", wantIdentifier, space.identifier)
	}

	var (
		identifier string
		name       string
	)
	if err := tx.QueryRowContext(
		ctx,
		`SELECT identifier, name FROM spaces WHERE id = $1`,
		string(space.id),
	).Scan(&identifier, &name); err != nil {
		t.Fatalf("スペースの取得に失敗: %v", err)
	}
	if identifier != wantIdentifier {
		t.Errorf("スペース識別子が %q であることを期待したが %q だった", wantIdentifier, identifier)
	}
	if name != wantName {
		t.Errorf("スペース名が %q であることを期待したが %q だった", wantName, name)
	}
}

// assertSpaceMemberCount checks how many accounts have joined the space.
//
// [Ja] assertSpaceMemberCount は、スペースに参加しているアカウント数を確認する。
func assertSpaceMemberCount(ctx context.Context, t *testing.T, tx *sql.Tx, spaceID model.SpaceID, want int) {
	t.Helper()

	var got int
	err := tx.QueryRowContext(
		ctx,
		`SELECT count(*) FROM space_members WHERE space_id = $1 AND active = true`,
		string(spaceID),
	).Scan(&got)
	if err != nil {
		t.Fatalf("スペースメンバー数の取得に失敗: %v", err)
	}
	if got != want {
		t.Errorf("スペースメンバーが %d 件であることを期待したが %d 件だった", want, got)
	}
}

// assertSpaceMemberScopes checks that the stored scopes are exactly the ones
// the seed intended to grant.
//
// [Ja] assertSpaceMemberScopes は、保存されたスコープがシードの与えようとした
// ものと完全に一致することを確認する。
func assertSpaceMemberScopes(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	label string,
	spaceID model.SpaceID,
	memberID model.SpaceMemberID,
	want []model.Scope,
) {
	t.Helper()

	var stored []string
	err := tx.QueryRowContext(
		ctx,
		`SELECT scopes FROM space_members WHERE id = $1 AND space_id = $2`,
		string(memberID),
		string(spaceID),
	).Scan(pq.Array(&stored))
	if err != nil {
		t.Fatalf("%sのスコープの取得に失敗: %v", label, err)
	}
	assertScopesEqual(t, label, stored, want)
}

// assertScopesEqual compares stored scopes against the intended set, ignoring
// order.
//
// [Ja] assertScopesEqual は、保存されたスコープを意図した集合と、順序を無視して
// 比較する。
func assertScopesEqual(t *testing.T, label string, stored []string, want []model.Scope) {
	t.Helper()

	got := make(map[string]bool, len(stored))
	for _, scope := range stored {
		got[scope] = true
	}
	if len(got) != len(want) {
		t.Errorf("%sのスコープが %d 件であることを期待したが %d 件だった (%v)", label, len(want), len(got), stored)
	}
	for _, scope := range want {
		if !got[string(scope)] {
			t.Errorf("%sにスコープ %s が付与されていない", label, scope)
		}
	}
}

// The Rails side still owns the creation of spaces and topics, and both
// Space::NAME_MAX_LENGTH and Topic::NAME_MAX_LENGTH are 30 there. seed-long-name
// carries names of exactly that length, in its own name and in the names of its
// topics, so that the wrapping it exists to expose is the wrapping of the
// longest name the model will accept. A name one character short would still
// look long while leaving that case unchecked.
//
// [Ja] スペースとトピックの作成を今も担当しているのは Rails 側で、そこでは
// Space::NAME_MAX_LENGTH も Topic::NAME_MAX_LENGTH も 30 である。seed-long-name は
// 自身の名前とトピックの名前の双方で、ちょうどその長さの名前を持つ。これにより、この
// スペースが表に出すための折り返しが、モデルが受け付ける最長の名前の折り返しになる。
// 1 文字短い名前でも長くは見えるが、そのケースは確認されないまま残る。
const seedLongNameMaxLength = 30

func TestSeedLongNameUsesTheFullNameLength(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		label string
		name  string
	}{
		{label: "スペース名", name: longNameSpaceName},
		{label: "トピック名 (日本語)", name: topicNameLongJapanese},
		{label: "トピック名 (半角英字)", name: topicNameLongASCII},
	} {
		if got := utf8.RuneCountInString(tt.name); got != seedLongNameMaxLength {
			t.Errorf("%sが %d 文字であることを期待したが %d 文字だった: %q", tt.label, seedLongNameMaxLength, got, tt.name)
		}
	}

	// The ASCII topic name is here for having nowhere to break, which is the one
	// case the Japanese name cannot cover. A space, hyphen or underscore that
	// slipped into it would give the layout a place to wrap and quietly turn it
	// into an ordinary long name.
	//
	// [Ja] 半角英字のトピック名は、折り返せる場所が無いという、日本語の名前では
	// 賄えない唯一のケースのために置いている。空白・ハイフン・アンダースコアが紛れ
	// 込むとレイアウトに折り返す場所を与えてしまい、ただ長いだけの名前に黙って
	// 変わってしまう。
	if strings.ContainsAny(topicNameLongASCII, " -_") {
		t.Errorf("トピック名 %q に折り返せる区切り文字が含まれている", topicNameLongASCII)
	}
}

// The demo space is the one place in the seed whose screens become pictures on
// the help pages, where a reader has no way to tell scaffolding from the
// product. Anything naming it that reaches those screens — the identifier in
// every page URL, the space name in the header, the topic name in the listing —
// therefore has to read as a wiki someone keeps rather than as seed data.
//
// The check is for the word rather than the prefix: seed- is how the other
// spaces are marked, and it is the wording that would give this one away
// wherever in a name it appeared.
//
// [Ja] デモスペースは、シードの中で唯一その画面がヘルプページの画像になる場所で
// あり、読者には足場とプロダクトを見分ける手立てが無い。したがって、それらの画面へ
// 届くもの、すなわちすべてのページ URL に入る識別子・ヘッダーに出るスペース名・
// 一覧に出るトピック名のいずれも、シードのデータではなく誰かが持っている Wiki と
// して読まれる必要がある。
//
// 接頭辞ではなく語そのものを見るのは、seed- が他のスペースの目印である一方、この
// スペースの正体を明かしてしまうのは、名前のどこに現れたとしてもその語のほうで
// あるため。
func TestDemoSpaceNamesCarryNoSeedWording(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		label string
		name  string
	}{
		{label: "スペース識別子", name: demoSpaceIdentifier},
		{label: "スペース名", name: demoSpaceName},
		{label: "トピック名", name: topicNameDemoMemo},
	} {
		if strings.Contains(strings.ToLower(tt.name), "seed") {
			t.Errorf("デモスペースの%sに seed が含まれている: %q", tt.label, tt.name)
		}
		if strings.Contains(tt.name, "シード") {
			t.Errorf("デモスペースの%sに シード が含まれている: %q", tt.label, tt.name)
		}
	}
}
