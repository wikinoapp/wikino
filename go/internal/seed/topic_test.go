package seed

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"testing"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGenerateTopics(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	spaces := buildSeedSpaces(t, tx, "seed-topics")

	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	for _, tt := range []struct {
		topic           *seededTopic
		wantName        string
		wantDescription string
		wantVisibility  model.TopicVisibility
		wantNumber      int32
		wantMembers     int
	}{
		{topic: topics.handbook, wantName: "ハンドブック", wantDescription: "一覧をページ送りできるだけのページを置いたトピックです。", wantVisibility: model.TopicVisibilityPublic, wantNumber: 1, wantMembers: 2},
		{topic: topics.notes, wantName: "ノート", wantDescription: "Markdown 記法・Wiki リンク・ページが取りうる状態を確認するためのトピックです。", wantVisibility: model.TopicVisibilityPublic, wantNumber: 2, wantMembers: 2},
		{topic: topics.sandbox, wantName: "サンドボックス", wantDescription: "表示が崩れやすい極端なページを置いたトピックです。", wantVisibility: model.TopicVisibilityPublic, wantNumber: 3, wantMembers: 2},
		{topic: topics.privateNotes, wantName: "非公開ノート", wantDescription: "両方のアカウントが参加している非公開トピックです。", wantVisibility: model.TopicVisibilityPrivate, wantNumber: 4, wantMembers: 2},
		{topic: topics.secret, wantName: "シークレット", wantDescription: "シードユーザー 1 だけが参加している非公開トピックです。", wantVisibility: model.TopicVisibilityPrivate, wantNumber: 5, wantMembers: 1},
	} {
		if tt.topic == nil {
			t.Errorf("トピック %s が結果に含まれていない", tt.wantName)

			continue
		}
		if tt.topic.name != tt.wantName {
			t.Errorf("トピック名が %q であることを期待したが %q だった", tt.wantName, tt.topic.name)
		}
		if tt.topic.spaceID != spaces.wiki.id {
			t.Errorf("トピック %s のスペースIDがseed-wikiと一致しない", tt.wantName)
		}
		assertTopicRow(ctx, t, tx, tt.topic, tt.wantName, tt.wantDescription, tt.wantVisibility, tt.wantNumber)
		assertTopicMemberCount(ctx, t, tx, tt.topic, tt.wantMembers)
	}

	// topics.secret is the one case that shows a private topic staying
	// hidden after joining the space, so roleCollaborator must not hold a
	// membership on it.
	//
	// [Ja] 「シークレット」トピックは、スペースに参加したあとも非公開トピックが見えない
	// ままであることを示す唯一のケースであるため、roleCollaborator がそこに
	// メンバーシップを持っていてはならない。
	assertTopicMemberScopes(ctx, t, tx, "「シークレット」のcollaborator", topics.secret, spaces.wiki.member(roleCollaborator), nil, false)

	assertTopicMemberScopes(ctx, t, tx, "「ハンドブック」のowner", topics.handbook, spaces.wiki.member(roleOwner), nil, true)
	assertTopicMemberScopes(ctx, t, tx, "「ハンドブック」のcollaborator", topics.handbook, spaces.wiki.member(roleCollaborator), nil, true)
	assertTopicMemberScopes(ctx, t, tx, "「シークレット」のowner", topics.secret, spaces.wiki.member(roleOwner), nil, true)

	// roleCollaborator sees topics.privateNotes only because the membership
	// itself carries topic:read: the space membership deliberately does not.
	//
	// [Ja] roleCollaborator が「非公開ノート」を見られるのは、メンバーシップ自身が
	// topic:read を持つからに他ならない。スペースメンバーシップは意図的にそれを
	// 持っていない。
	assertTopicMemberScopes(
		ctx, t, tx, "「非公開ノート」のcollaborator",
		topics.privateNotes, spaces.wiki.member(roleCollaborator), []model.Scope{model.ScopeTopicRead}, true,
	)

	assertSoloTopics(ctx, t, tx, spaces.solo, topics)
}

func TestCreateTopicRejectsRoleWithoutSpaceMembership(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	spaces := buildSeedSpaces(t, tx, "seed-topic-role-guard")
	spec := topicSpec{
		name:        "Invalid Collaborator Topic",
		description: "Must not be inserted without a collaborator space membership.",
		visibility:  model.TopicVisibilityPublic,
		memberRoles: []seedRole{roleOwner, roleCollaborator},
	}

	_, err := createTopic(ctx, tx, spaces.solo, spec, 1)
	wantErr := "トピック Invalid Collaborator Topic は役割 collaborator の参加を指定しているが、その役割はスペースに参加していない"
	if err == nil {
		t.Fatal("指定した役割がスペースに参加していない場合にエラーを期待したがnilだった")
	}
	if err.Error() != wantErr {
		t.Errorf("エラーが %q であることを期待したが %q だった", wantErr, err)
	}

	var count int
	if err := tx.QueryRowContext(
		ctx,
		`SELECT count(*) FROM topics WHERE space_id = $1 AND name = $2`,
		string(spaces.solo.id),
		spec.name,
	).Scan(&count); err != nil {
		t.Fatalf("トピック数の取得に失敗: %v", err)
	}
	if count != 0 {
		t.Errorf("不正なトピックが作成されないことを期待したが %d 件あった", count)
	}
}

func TestTopicVisibilityForSeededMembers(t *testing.T) {
	t.Parallel()

	privateTopic := &model.Topic{Visibility: model.TopicVisibilityPrivate}
	publicTopic := &model.Topic{Visibility: model.TopicVisibilityPublic}

	admin := &seededSpaceMember{scopes: adminSpaceScopes}
	collaborator := &seededSpaceMember{scopes: nonAdminSpaceScopes}

	// These four are the states the topics of seed-wiki are arranged to
	// produce. If the scope sets ever drift, the topics stop demonstrating
	// what they were put there for, and the seed goes on running without
	// saying so.
	//
	// [Ja] この 4 つが、seed-wiki のトピックが作り出そうとしている状態。スコープの
	// 組み合わせがずれると、トピックは置かれた目的を示さなくなるが、シードは
	// それを告げずに動き続けてしまう。
	tests := []struct {
		name       string
		topic      *model.Topic
		member     *seededSpaceMember
		wantCanSee bool
	}{
		{name: "管理者は参加した非公開トピックを見られる", topic: privateTopic, member: admin, wantCanSee: true},
		{name: "非管理者も参加した非公開トピックを見られる", topic: privateTopic, member: collaborator, wantCanSee: true},
		{name: "公開トピックはスコープ無しでも見られる", topic: publicTopic, member: collaborator, wantCanSee: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scopes := topicMemberScopes(tt.topic.Visibility, tt.member)
			got := policy.NewMemberPolicy(tt.member.scopes, scopes).CanShowTopic(tt.topic)
			if got != tt.wantCanSee {
				t.Errorf("トピックの閲覧可否が %t であることを期待したが %t だった", tt.wantCanSee, got)
			}
		})
	}

	// Not joining a private topic has to be enough to keep it hidden. The
	// non-admin scope set is what makes that true, so it is checked here
	// rather than assumed.
	//
	// [Ja] 非公開トピックに参加していないことだけで、それが隠れたままである必要が
	// ある。それを成り立たせているのは非管理者のスコープ集合であるため、前提に
	// せずここで確認する。
	if policy.NewMemberPolicy(collaborator.scopes, nil).CanShowTopic(privateTopic) {
		t.Error("非管理者は参加していない非公開トピックを見られないことを期待したが見られた")
	}
}

// buildSeedSpaces assembles the spaces generateTopics needs without calling
// generateSpaces, which fixes both space identifiers: two tests creating them
// at once would wait on each other at the unique index.
//
// [Ja] buildSeedSpaces は generateTopics が必要とするスペースを、generateSpaces を
// 呼ばずに組み立てる。generateSpaces は 2 つのスペース識別子を固定するため、
// それを同時に作る 2 つのテストは一意インデックスの上で待ち合わせてしまう。
func buildSeedSpaces(t *testing.T, tx *sql.Tx, prefix string) *seededSpaces {
	t.Helper()

	users := buildSeedUsers(t, tx, prefix)

	build := func(suffix string, memberSpecs []spaceMemberSpec) *seededSpace {
		identifier := prefix + "-" + suffix
		spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier(identifier).Build()
		space := &seededSpace{
			id:         spaceID,
			identifier: model.SpaceIdentifier(identifier),
			members:    make(map[seedRole]*seededSpaceMember, len(memberSpecs)),
		}
		for _, memberSpec := range memberSpecs {
			space.members[memberSpec.role] = &seededSpaceMember{
				id: testutil.NewSpaceMemberBuilder(t, tx).
					WithSpaceID(spaceID).
					WithUserID(users.user(memberSpec.role).ID).
					WithScopes(memberSpec.scopes).
					Build(),
				scopes: memberSpec.scopes,
			}
		}

		return space
	}

	return &seededSpaces{
		wiki: build("wiki", []spaceMemberSpec{
			{role: roleOwner, scopes: adminSpaceScopes},
			{role: roleCollaborator, scopes: nonAdminSpaceScopes},
			{role: roleGuest, scopes: nonAdminSpaceScopes},
		}),
		solo: build("solo", []spaceMemberSpec{{role: roleOwner, scopes: adminSpaceScopes}}),
	}
}

// assertTopicRow checks the stored display text, visibility and number of a
// topic. The number decides where the topic sits in the listing and is part of
// its URL.
//
// [Ja] assertTopicRow は、保存されたトピックの表示テキスト・公開範囲・番号を
// 確認する。番号は一覧内での位置を決め、URL の一部にもなる。
func assertTopicRow(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	topic *seededTopic,
	wantName string,
	wantDescription string,
	wantVisibility model.TopicVisibility,
	wantNumber int32,
) {
	t.Helper()

	var (
		name        string
		description string
		visibility  int32
		number      int32
	)
	err := tx.QueryRowContext(
		ctx,
		`SELECT name, description, visibility, number FROM topics WHERE space_id = $1 AND id = $2`,
		string(topic.spaceID), string(topic.id),
	).Scan(&name, &description, &visibility, &number)
	if err != nil {
		t.Fatalf("トピック %s の取得に失敗: %v", wantName, err)
	}

	if name != wantName {
		t.Errorf("トピック名が %q であることを期待したが %q だった", wantName, name)
	}
	if description != wantDescription {
		t.Errorf("トピック %s の説明が %q であることを期待したが %q だった", wantName, wantDescription, description)
	}
	if model.TopicVisibility(visibility) != wantVisibility {
		t.Errorf("トピック %s の公開範囲が %d であることを期待したが %d だった", wantName, wantVisibility, visibility)
	}
	if number != wantNumber {
		t.Errorf("トピック %s の番号が %d であることを期待したが %d だった", wantName, wantNumber, number)
	}
}

// assertTopicMemberCount checks how many accounts have joined the topic.
//
// [Ja] assertTopicMemberCount は、トピックに参加しているアカウント数を確認する。
func assertTopicMemberCount(ctx context.Context, t *testing.T, tx *sql.Tx, topic *seededTopic, want int) {
	t.Helper()

	var got int
	err := tx.QueryRowContext(
		ctx,
		`SELECT count(*) FROM topic_members WHERE space_id = $1 AND topic_id = $2`,
		string(topic.spaceID), string(topic.id),
	).Scan(&got)
	if err != nil {
		t.Fatalf("トピック %s のメンバー数の取得に失敗: %v", topic.name, err)
	}
	if got != want {
		t.Errorf("トピック %s のメンバーが %d 件であることを期待したが %d 件だった", topic.name, want, got)
	}
}

// assertTopicMemberScopes checks whether the space member has joined the topic
// and, when it has, that the membership carries exactly the given scopes.
//
// [Ja] assertTopicMemberScopes は、スペースメンバーがトピックに参加しているかと、
// 参加している場合にそのメンバーシップが与えられたスコープと完全に一致することを
// 確認する。
func assertTopicMemberScopes(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	label string,
	topic *seededTopic,
	member *seededSpaceMember,
	want []model.Scope,
	wantJoined bool,
) {
	t.Helper()

	var stored []string
	err := tx.QueryRowContext(
		ctx,
		`SELECT scopes FROM topic_members WHERE space_id = $1 AND topic_id = $2 AND space_member_id = $3`,
		string(topic.spaceID), string(topic.id), string(member.id),
	).Scan(pq.Array(&stored))

	if !wantJoined {
		if err == nil {
			t.Errorf("%sは参加していないことを期待したがメンバーシップが存在した", label)
		} else if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("%sのトピックメンバーの取得に失敗: %v", label, err)
		}

		return
	}
	if err != nil {
		t.Fatalf("%sのトピックメンバーの取得に失敗: %v", label, err)
	}
	assertScopesEqual(t, label, stored, want)
}

// assertSoloTopics checks the topics of seed-solo, whose point is that
// roleCollaborator is not a member of the space at all: the public one is
// listed to that account and the private one is not.
//
// [Ja] assertSoloTopics は seed-solo のトピックを確認する。このスペースの要点は
// roleCollaborator がそもそもスペースのメンバーでないことにあり、公開トピックは
// そのアカウントにも一覧され、非公開トピックは一覧されない。
func assertSoloTopics(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	solo *seededSpace,
	topics *seededTopics,
) {
	t.Helper()

	// The seed-solo topics are carried on the result because the page generator
	// writes into them. A topic left unassigned would be created in the
	// database and then be unreachable from the generator that fills it.
	//
	// [Ja] seed-solo のトピックは、ページの生成器が書き込むため結果に載せている。
	// 割り当てられなかったトピックは、データベースには作成されるものの、そこを
	// 埋める生成器からは辿れなくなる。
	for _, tt := range []struct {
		topic    *seededTopic
		wantName string
	}{
		{topic: topics.soloNotes, wantName: topicNameSoloNotes},
		{topic: topics.soloSecret, wantName: topicNameSoloSecret},
	} {
		if tt.topic == nil {
			t.Errorf("トピック %s が結果に含まれていない", tt.wantName)

			continue
		}
		if tt.topic.name != tt.wantName {
			t.Errorf("トピック名が %q であることを期待したが %q だった", tt.wantName, tt.topic.name)
		}
		if tt.topic.spaceID != solo.id {
			t.Errorf("トピック %s のスペースIDがseed-soloと一致しない", tt.wantName)
		}
	}

	rows, err := tx.QueryContext(
		ctx,
		`SELECT name, description, visibility FROM topics WHERE space_id = $1 ORDER BY number`,
		string(solo.id),
	)
	if err != nil {
		t.Fatalf("seed-soloのトピック取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	type topicState struct {
		description string
		visibility  model.TopicVisibility
	}
	got := make(map[string]topicState)
	for rows.Next() {
		var (
			name        string
			description string
			visibility  int32
		)
		if err := rows.Scan(&name, &description, &visibility); err != nil {
			t.Fatalf("seed-soloのトピックの読み取りに失敗: %v", err)
		}
		got[name] = topicState{description: description, visibility: model.TopicVisibility(visibility)}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("seed-soloのトピックの走査に失敗: %v", err)
	}

	want := map[string]topicState{
		"個人ノート": {
			description: "スペースを開いた人なら誰にでも一覧に出る公開トピックです。",
			visibility:  model.TopicVisibilityPublic,
		},
		"個人シークレット": {
			description: "スペースの外からは見えない非公開トピックです。",
			visibility:  model.TopicVisibilityPrivate,
		},
	}
	if len(got) != len(want) {
		t.Errorf("seed-soloのトピックが %d 件であることを期待したが %d 件だった", len(want), len(got))
	}
	for name, wantState := range want {
		state, exists := got[name]
		if !exists {
			t.Errorf("seed-soloにトピック %s が無い", name)

			continue
		}
		if state.description != wantState.description {
			t.Errorf("seed-soloのトピック %s の説明が %q であることを期待したが %q だった", name, wantState.description, state.description)
		}
		if state.visibility != wantState.visibility {
			t.Errorf("seed-soloのトピック %s の公開範囲が %d であることを期待したが %d だった", name, wantState.visibility, state.visibility)
		}
		// A non-member is judged by GuestPolicy, which lets the public topic
		// through and stops the private one.
		//
		// [Ja] 非メンバーの判定は GuestPolicy が行い、公開トピックは通し、非公開
		// トピックは止める。
		canSee := policy.NewGuestPolicy().CanShowTopic(&model.Topic{Visibility: state.visibility})
		if canSee != (state.visibility == model.TopicVisibilityPublic) {
			t.Errorf("非メンバーから見たトピック %s の閲覧可否が期待と異なる", name)
		}
	}
}
