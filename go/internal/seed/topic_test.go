package seed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

	// 非公開トピックの説明文は、そこに参加しているアカウントを名指しするため、
	// 期待値は書き下さずに同じメンバーシップから組み立てる。そうすることで確認できる
	// のは、説明文がそのトピックのアカウントを名指ししていることになる。文をテストへ
	// 書き写しても、同じ文字列を2箇所へ書いたことしか確認できない。
	ownerName := spaces.wiki.member(roleOwner).name
	collaboratorName := spaces.wiki.member(roleCollaborator).name

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
		{topic: topics.privateNotes, wantName: "非公開ノート", wantDescription: fmt.Sprintf("%s と %s が参加している非公開トピックです。", ownerName, collaboratorName), wantVisibility: model.TopicVisibilityPrivate, wantNumber: 4, wantMembers: 2},
		{topic: topics.secret, wantName: "シークレット", wantDescription: fmt.Sprintf("%s だけが参加している非公開トピックです。", ownerName), wantVisibility: model.TopicVisibilityPrivate, wantNumber: 5, wantMembers: 1},
		{topic: topics.export, wantName: "エクスポート", wantDescription: "エクスポートした ZIP で、ページタイトルがどうファイル名になるかを確認するためのトピックです。記号の置き換え・Windows の予約名・大文字小文字だけが違う名前の衝突・Wiki リンクの書き換えを見ます。", wantVisibility: model.TopicVisibilityPublic, wantNumber: 6, wantMembers: 2},
		{topic: topics.exportSymbol, wantName: "エクスポート*記号", wantDescription: "名前の半角記号が ZIP の中でディレクトリ名「エクスポート＊記号」に変わることを、名前自体で確認するためのトピックです。frontmatter を持つ本文の扱いを見るページを置きます。", wantVisibility: model.TopicVisibilityPublic, wantNumber: 7, wantMembers: 2},
	} {
		if tt.topic == nil {
			t.Errorf("トピック%sが結果に含まれていない", tt.wantName)

			continue
		}
		if tt.topic.name != tt.wantName {
			t.Errorf("トピック名が%qであることを期待したが%qだった", tt.wantName, tt.topic.name)
		}
		if tt.topic.spaceID != spaces.wiki.id {
			t.Errorf("トピック%sのスペースIDがseed-wikiと一致しない", tt.wantName)
		}
		assertTopicRow(ctx, t, tx, tt.topic, tt.wantName, tt.wantDescription, tt.wantVisibility, tt.wantNumber)
		assertTopicMemberCount(ctx, t, tx, tt.topic, tt.wantMembers)
	}

	// 「シークレット」トピックは、スペースに参加したあとも非公開トピックが見えない
	// ままであることを示す唯一のケースであるため、roleCollaboratorがそこに
	// メンバーシップを持っていてはならない。
	assertTopicMemberScopes(ctx, t, tx, "「シークレット」のcollaborator", topics.secret, spaces.wiki.member(roleCollaborator), nil, false)

	assertTopicMemberScopes(ctx, t, tx, "「ハンドブック」のowner", topics.handbook, spaces.wiki.member(roleOwner), nil, true)
	assertTopicMemberScopes(ctx, t, tx, "「ハンドブック」のcollaborator", topics.handbook, spaces.wiki.member(roleCollaborator), nil, true)
	assertTopicMemberScopes(ctx, t, tx, "「シークレット」のowner", topics.secret, spaces.wiki.member(roleOwner), nil, true)

	// roleCollaboratorが「非公開ノート」を見られるのは、メンバーシップ自身が
	// topic:readを持つからに他ならない。スペースメンバーシップは意図的にそれを
	// 持っていない。
	assertTopicMemberScopes(
		ctx, t, tx, "「非公開ノート」のcollaborator",
		topics.privateNotes, spaces.wiki.member(roleCollaborator), []model.Scope{model.ScopeTopicRead}, true,
	)

	// エクスポートの2つのトピックには、上の公開トピックと同じ2つのアカウントが
	// 参加している。メンバー数に頼らずここで両方の役割を名指しすることが、その
	// アカウントがownerとcollaboratorであることを示す。2件という数は、どちらかへ
	// guestを参加させても同じになる。
	assertTopicMemberScopes(ctx, t, tx, "「エクスポート」のowner", topics.export, spaces.wiki.member(roleOwner), nil, true)
	assertTopicMemberScopes(ctx, t, tx, "「エクスポート」のcollaborator", topics.export, spaces.wiki.member(roleCollaborator), nil, true)
	assertTopicMemberScopes(ctx, t, tx, "「エクスポート*記号」のowner", topics.exportSymbol, spaces.wiki.member(roleOwner), nil, true)
	assertTopicMemberScopes(ctx, t, tx, "「エクスポート*記号」のcollaborator", topics.exportSymbol, spaces.wiki.member(roleCollaborator), nil, true)

	// 「エクスポート*記号」の名前は、半角の "*" を保ったままデータベースへ届く
	// 必要がある。この文字こそがこのトピックの目的であり、エクスポートはディレクトリ名を
	// 付けるときにこれを "＊" へ変える。あらかじめ全角で保存された名前では、変換が
	// 一度も動かないままアーカイブが正しく見えてしまう。上のテーブル駆動の行も名前を
	// 行から読んではいるが、そこでは全トピックの他のフィールドと並んでおり、半角の "*"
	// が残ること自体が確認の対象だとは読めない。独立した確認として置くのはそのためである。
	var storedSymbolName string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT name FROM topics WHERE space_id = $1 AND id = $2`,
		string(topics.exportSymbol.spaceID), string(topics.exportSymbol.id),
	).Scan(&storedSymbolName); err != nil {
		t.Fatalf("トピック%sの名前の取得に失敗: %v", topicNameExportSymbol, err)
	}
	if storedSymbolName != "エクスポート*記号" {
		t.Errorf("記号を含むトピック名が%qのまま保存されることを期待したが%qだった", "エクスポート*記号", storedSymbolName)
	}

	assertSoloTopics(ctx, t, tx, spaces.solo, topics)
	assertLongNameTopics(ctx, t, tx, spaces.longName)
	assertDemoTopic(ctx, t, tx, spaces.demo, topics)
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
	wantErr := "トピック Invalid Collaborator Topicは役割 collaboratorの参加を指定しているが、その役割はスペースに参加していない"
	if err == nil {
		t.Fatal("指定した役割がスペースに参加していない場合にエラーを期待したがnilだった")
	}
	if err.Error() != wantErr {
		t.Errorf("エラーが%qであることを期待したが%qだった", wantErr, err)
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
		t.Errorf("不正なトピックが作成されないことを期待したが%d件あった", count)
	}
}

func TestTopicVisibilityForSeededMembers(t *testing.T) {
	t.Parallel()

	privateTopic := &model.Topic{Visibility: model.TopicVisibilityPrivate}
	publicTopic := &model.Topic{Visibility: model.TopicVisibilityPublic}

	admin := &seededSpaceMember{scopes: adminSpaceScopes}
	collaborator := &seededSpaceMember{scopes: nonAdminSpaceScopes}

	// この4つが、seed-wikiのトピックが作り出そうとしている状態。スコープの
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
				t.Errorf("トピックの閲覧可否が%tであることを期待したが%tだった", tt.wantCanSee, got)
			}
		})
	}

	// 非公開トピックに参加していないことだけで、それが隠れたままである必要が
	// ある。それを成り立たせているのは非管理者のスコープ集合であるため、前提に
	// せずここで確認する。
	if policy.NewMemberPolicy(collaborator.scopes, nil).CanShowTopic(privateTopic) {
		t.Error("非管理者は参加していない非公開トピックを見られないことを期待したが見られた")
	}
}

// buildSeedSpacesはgenerateTopicsが必要とするスペースを、generateSpacesを
// 呼ばずに組み立てる。generateSpacesはスペース識別子を固定するため、それを同時に
// 作る2つのテストは一意インデックスの上で待ち合わせてしまう。
func buildSeedSpaces(t *testing.T, tx *sql.Tx, prefix string) *seededSpaces {
	t.Helper()

	_, spaces := buildSeedUsersAndSpaces(t, tx, prefix)

	return spaces
}

// buildSeedUsersAndSpacesは同じことを行い、アカウントも併せて返す。自身が
// 書き込むスペースに参加していないアカウントを名指しする生成器のためのもの。その種の
// 生成器はスペースではなくアカウントを受け取るため、それを確認するテストには両方が
// 揃っている必要がある。
func buildSeedUsersAndSpaces(t *testing.T, tx *sql.Tx, prefix string) (*seededUsers, *seededSpaces) {
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
				name:   users.user(memberSpec.role).Name,
				scopes: memberSpec.scopes,
			}
		}

		return space
	}

	return users, &seededSpaces{
		wiki: build("wiki", []spaceMemberSpec{
			{role: roleOwner, scopes: adminSpaceScopes},
			{role: roleCollaborator, scopes: nonAdminSpaceScopes},
			{role: roleGuest, scopes: nonAdminSpaceScopes},
		}),
		solo:     build("solo", []spaceMemberSpec{{role: roleOwner, scopes: adminSpaceScopes}}),
		longName: build("long", []spaceMemberSpec{{role: roleOwner, scopes: adminSpaceScopes}}),
		demo:     build("demo", []spaceMemberSpec{{role: roleOwner, scopes: adminSpaceScopes}}),
	}
}

// assertTopicRowは、保存されたトピックの表示テキスト・公開範囲・番号を
// 確認する。番号は一覧内での位置を決め、URLの一部にもなる。
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
		t.Fatalf("トピック%sの取得に失敗: %v", wantName, err)
	}

	if name != wantName {
		t.Errorf("トピック名が%qであることを期待したが%qだった", wantName, name)
	}
	if description != wantDescription {
		t.Errorf("トピック%sの説明が%qであることを期待したが%qだった", wantName, wantDescription, description)
	}
	if model.TopicVisibility(visibility) != wantVisibility {
		t.Errorf("トピック%sの公開範囲が%dであることを期待したが%dだった", wantName, wantVisibility, visibility)
	}
	if number != wantNumber {
		t.Errorf("トピック%sの番号が%dであることを期待したが%dだった", wantName, wantNumber, number)
	}
}

// assertTopicMemberCountは、トピックに参加しているアカウント数を確認する。
func assertTopicMemberCount(ctx context.Context, t *testing.T, tx *sql.Tx, topic *seededTopic, want int) {
	t.Helper()

	var got int
	err := tx.QueryRowContext(
		ctx,
		`SELECT count(*) FROM topic_members WHERE space_id = $1 AND topic_id = $2`,
		string(topic.spaceID), string(topic.id),
	).Scan(&got)
	if err != nil {
		t.Fatalf("トピック%sのメンバー数の取得に失敗: %v", topic.name, err)
	}
	if got != want {
		t.Errorf("トピック%sのメンバーが%d件であることを期待したが%d件だった", topic.name, want, got)
	}
}

// assertTopicMemberScopesは、スペースメンバーがトピックに参加しているかと、
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

// assertSoloTopicsはseed-soloのトピックを確認する。このスペースの要点は
// roleCollaboratorがそもそもスペースのメンバーでないことにあり、公開トピックは
// そのアカウントにも一覧され、非公開トピックは一覧されない。
func assertSoloTopics(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	solo *seededSpace,
	topics *seededTopics,
) {
	t.Helper()

	// seed-soloのトピックは、ページの生成器が書き込むため結果に載せている。
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
			t.Errorf("トピック%sが結果に含まれていない", tt.wantName)

			continue
		}
		if tt.topic.name != tt.wantName {
			t.Errorf("トピック名が%qであることを期待したが%qだった", tt.wantName, tt.topic.name)
		}
		if tt.topic.spaceID != solo.id {
			t.Errorf("トピック%sのスペースIDがseed-soloと一致しない", tt.wantName)
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
		t.Errorf("seed-soloのトピックが%d件であることを期待したが%d件だった", len(want), len(got))
	}
	for name, wantState := range want {
		state, exists := got[name]
		if !exists {
			t.Errorf("seed-soloにトピック%sが無い", name)

			continue
		}
		if state.description != wantState.description {
			t.Errorf("seed-soloのトピック%sの説明が%qであることを期待したが%qだった", name, wantState.description, state.description)
		}
		if state.visibility != wantState.visibility {
			t.Errorf("seed-soloのトピック%sの公開範囲が%dであることを期待したが%dだった", name, wantState.visibility, state.visibility)
		}
		// 非メンバーの判定はGuestPolicyが行い、公開トピックは通し、非公開
		// トピックは止める。
		canSee := policy.NewGuestPolicy().CanShowTopic(&model.Topic{Visibility: state.visibility})
		if canSee != (state.visibility == model.TopicVisibilityPublic) {
			t.Errorf("非メンバーから見たトピック%sの閲覧可否が期待と異なる", name)
		}
	}
}

// assertLongNameTopicsはseed-long-nameのトピックを確認する。結果ではなく
// データベースから読むのは、これらへページを書き込む生成器が無く、結果へ割り当てて
// いないため。満たされるべきなのは、画面が描くことになる名前を持った行がそのスペース
// の下に存在することと、他のスペースのトピックと同じく1から採番されていることである。
func assertLongNameTopics(ctx context.Context, t *testing.T, tx *sql.Tx, longName *seededSpace) {
	t.Helper()

	rows, err := tx.QueryContext(
		ctx,
		`SELECT name, description, visibility, number FROM topics WHERE space_id = $1 ORDER BY number`,
		string(longName.id),
	)
	if err != nil {
		t.Fatalf("seed-long-nameのトピック取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	type topicState struct {
		description string
		visibility  model.TopicVisibility
		number      int32
	}
	got := make(map[string]topicState)
	for rows.Next() {
		var (
			name        string
			description string
			visibility  int32
			number      int32
		)
		if err := rows.Scan(&name, &description, &visibility, &number); err != nil {
			t.Fatalf("seed-long-nameのトピックの読み取りに失敗: %v", err)
		}
		got[name] = topicState{description: description, visibility: model.TopicVisibility(visibility), number: number}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("seed-long-nameのトピックの走査に失敗: %v", err)
	}

	want := map[string]topicState{
		topicNameLongJapanese: {
			description: "名前を最大文字数まで伸ばした公開トピックです。",
			visibility:  model.TopicVisibilityPublic,
			number:      1,
		},
		topicNameLongASCII: {
			description: "名前を、折り返せない半角英字だけで最大文字数まで伸ばした公開トピックです。",
			visibility:  model.TopicVisibilityPublic,
			number:      2,
		},
	}
	if len(got) != len(want) {
		t.Errorf("seed-long-nameのトピックが%d件であることを期待したが%d件だった", len(want), len(got))
	}
	for name, wantState := range want {
		state, exists := got[name]
		if !exists {
			t.Errorf("seed-long-nameにトピック%sが無い", name)

			continue
		}
		if state.description != wantState.description {
			t.Errorf("seed-long-nameのトピック%sの説明が%qであることを期待したが%qだった", name, wantState.description, state.description)
		}
		if state.visibility != wantState.visibility {
			t.Errorf("seed-long-nameのトピック%sの公開範囲が%dであることを期待したが%dだった", name, wantState.visibility, state.visibility)
		}
		if state.number != wantState.number {
			t.Errorf("seed-long-nameのトピック%sの番号が%dであることを期待したが%dだった", name, wantState.number, state.number)
		}
	}

	// これらのトピックに参加するのはroleOwnerだけであり、それがブラウザ確認で
	// サインインするアカウントの前にこれらを出す。メンバーシップ無しで作成された
	// トピックもスペース管理者には一覧に出るため、確認すべきなのは一覧ではなく
	// メンバーシップのほうになる。
	var memberCount int
	if err := tx.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM topic_members WHERE space_id = $1`,
		string(longName.id),
	).Scan(&memberCount); err != nil {
		t.Fatalf("seed-long-nameのトピックメンバー数の取得に失敗: %v", err)
	}
	if memberCount != len(want) {
		t.Errorf("seed-long-nameのトピックメンバーが%d件であることを期待したが%d件だった", len(want), memberCount)
	}
}

// assertDemoTopicはデモスペースのトピックを確認する。デモページはすべて
// そこへ書き込まれるため、データベースだけでなく結果からも読み取る。生成器が
// 辿れないトピックは、デモページの行き場が無いことを意味するため。
func assertDemoTopic(ctx context.Context, t *testing.T, tx *sql.Tx, demo *seededSpace, topics *seededTopics) {
	t.Helper()

	topic := topics.demoMemo
	if topic == nil {
		t.Fatalf("トピック%sが結果に含まれていない", topicNameDemoMemo)
	}
	if topic.name != topicNameDemoMemo {
		t.Errorf("トピック名が%qであることを期待したが%qだった", topicNameDemoMemo, topic.name)
	}
	if topic.spaceID != demo.id {
		t.Errorf("トピック%sのスペースIDがデモスペースと一致しない", topicNameDemoMemo)
	}

	assertTopicRow(
		ctx, t, tx, topic,
		topicNameDemoMemo,
		"日々の覚え書きです。行った場所や読んだもの、作ったものを書きためています。",
		model.TopicVisibilityPrivate,
		1,
	)
	assertTopicMemberCount(ctx, t, tx, topic, 1)
	assertTopicMemberScopes(ctx, t, tx, "Memoのowner", topic, demo.member(roleOwner), nil, true)

	// デモスペースが持つトピックはこれだけである。デモ本文のWikiリンクは
	// いずれもトピックを伴わずタイトルだけを名指ししており、その種のリンクは書かれた
	// トピックの中でしか解決しない。ここに2つ目のトピックがあると、デモページが
	// リンクの届かない場所へ置かれうることになる。
	var count int
	if err := tx.QueryRowContext(
		ctx,
		`SELECT count(*) FROM topics WHERE space_id = $1`,
		string(demo.id),
	).Scan(&count); err != nil {
		t.Fatalf("デモスペースのトピック数の取得に失敗: %v", err)
	}
	if count != 1 {
		t.Errorf("デモスペースのトピックが1件であることを期待したが%d件だった", count)
	}
}
