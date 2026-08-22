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

// Topic names are constants because later generators pick the topic they write
// into by name, and because the browser verification navigates by these names.
//
// [Ja] トピック名を定数にしているのは、後続の生成器が書き込み先のトピックを名前で
// 選ぶことと、ブラウザ確認がこの名前を辿ることによる。
const (
	topicNameHandbook     = "ハンドブック"
	topicNameNotes        = "ノート"
	topicNameSandbox      = "サンドボックス"
	topicNamePrivateNotes = "非公開ノート"
	topicNameSecret       = "シークレット"
	topicNameSoloNotes    = "個人ノート"
	topicNameSoloSecret   = "個人シークレット"
)

// topicSpec describes one topic to create. The specs are held as an ordered
// list so that the topic numbers, which are unique per space and decide the
// order of the listing, are counted off from the position rather than written
// out by hand.
//
// [Ja] topicSpec は作成するトピック 1 件の内容。仕様を順序付きの一覧として持つのは、
// スペース内で一意であり一覧の並び順を決めるトピック番号を、手書きせずに位置から
// 採番するため。
type topicSpec struct {
	name        string
	description string
	visibility  model.TopicVisibility
	// memberRoles are the accounts joined to the topic. Naming them here rather
	// than deriving them from the space is what keeps a topic only one role may
	// read from opening to an account added to the space later.
	//
	// [Ja] memberRoles はトピックに参加させるアカウント。スペースから導かずここで
	// 名指しするのは、1 つの役割だけが読めるトピックが、後からスペースへ足した
	// アカウントに開いてしまわないようにするため。
	memberRoles []seedRole
	// assign stores the created topic on the result, for the specs whose topic
	// a later generator writes pages into. It is nil for the rest.
	//
	// [Ja] assign は、後続の生成器がページを書き込むトピックについて、作成した
	// トピックを結果へ格納する。それ以外では nil。
	assign func(topics *seededTopics, topic *seededTopic)
}

// wikiTopicSpecs returns the topics of seed-wiki. Between them they cover what
// the topic screens can be in: public and private, joined by more than one
// account and by one alone, with and without a large number of pages.
//
// The specs are built from the space rather than held as a package-level list
// because the descriptions of the private topics name the accounts that joined
// them, and what those accounts are called is written in the roster. Reading
// the names off the memberships is what keeps the description of a topic and
// the accounts inside it saying the same thing.
//
// A name goes into a description as it is, unlike one going into a page body.
// A description reaches the screen as text, drawn through templ, which escapes
// it; nothing reads it as Markdown, so there is no notation in a name for the
// screen to act on. Encoding one the way markdownPlainText encodes a body would
// put the numeric character references themselves on the screen.
//
// [Ja] wikiTopicSpecs は seed-wiki のトピックを返す。全体で、トピック画面が取りうる
// 状態を網羅する。公開と非公開、複数のアカウントが参加しているものと 1 つだけが
// 参加しているもの、ページが大量にあるものと無いもの。
//
// 仕様をパッケージ変数の一覧ではなくスペースから組み立てるのは、非公開トピックの
// 説明文が、そこに参加しているアカウントを名指しするため。そのアカウントが何と
// 呼ばれるのかは名簿に書かれている。名前をメンバーシップから読むことが、トピックの
// 説明文と、その中にいるアカウントとが同じことを述べ続ける理由になる。
//
// 名前は説明文へそのまま入れる。ページ本文へ入れる場合とは違う。説明文は templ を
// 通ってテキストとして画面へ出て、templ がエスケープする。Markdown として読む経路が
// 無いため、名前に書かれた記法が画面で効くこともない。本文と同じく markdownPlainText
// でエンコードすると、数値文字参照そのものが画面に出る。
func wikiTopicSpecs(wiki *seededSpace) ([]topicSpec, error) {
	owner, err := wiki.requireMember(roleOwner)
	if err != nil {
		return nil, err
	}
	collaborator, err := wiki.requireMember(roleCollaborator)
	if err != nil {
		return nil, err
	}

	return []topicSpec{
		{
			name:        topicNameHandbook,
			description: "一覧をページ送りできるだけのページを置いたトピックです。",
			visibility:  model.TopicVisibilityPublic,
			memberRoles: []seedRole{roleOwner, roleCollaborator},
			assign:      func(topics *seededTopics, topic *seededTopic) { topics.handbook = topic },
		},
		{
			name:        topicNameNotes,
			description: "Markdown 記法・Wiki リンク・ページが取りうる状態を確認するためのトピックです。",
			visibility:  model.TopicVisibilityPublic,
			memberRoles: []seedRole{roleOwner, roleCollaborator},
			assign:      func(topics *seededTopics, topic *seededTopic) { topics.notes = topic },
		},
		{
			name:        topicNameSandbox,
			description: "表示が崩れやすい極端なページを置いたトピックです。",
			visibility:  model.TopicVisibilityPublic,
			memberRoles: []seedRole{roleOwner, roleCollaborator},
			assign:      func(topics *seededTopics, topic *seededTopic) { topics.sandbox = topic },
		},
		{
			name:        topicNamePrivateNotes,
			description: fmt.Sprintf("%s と %s が参加している非公開トピックです。", owner.name, collaborator.name),
			visibility:  model.TopicVisibilityPrivate,
			memberRoles: []seedRole{roleOwner, roleCollaborator},
			assign:      func(topics *seededTopics, topic *seededTopic) { topics.privateNotes = topic },
		},
		{
			name:        topicNameSecret,
			description: fmt.Sprintf("%s だけが参加している非公開トピックです。", owner.name),
			visibility:  model.TopicVisibilityPrivate,
			memberRoles: []seedRole{roleOwner},
			assign:      func(topics *seededTopics, topic *seededTopic) { topics.secret = topic },
		},
	}, nil
}

// soloTopicSpecs are the topics of seed-solo. Only roleOwner has joined that
// space, so these two show what a space looks like to someone outside it: the
// public topic is listed, the private one is not.
//
// [Ja] soloTopicSpecs は seed-solo のトピック。このスペースに参加しているのは
// roleOwner だけであるため、この 2 つでスペースを外から見た状態を確認できる。
// 公開トピックは一覧に出て、非公開トピックは出ない。
var soloTopicSpecs = []topicSpec{
	{
		name:        topicNameSoloNotes,
		description: "スペースを開いた人なら誰にでも一覧に出る公開トピックです。",
		visibility:  model.TopicVisibilityPublic,
		memberRoles: []seedRole{roleOwner},
		assign:      func(topics *seededTopics, topic *seededTopic) { topics.soloNotes = topic },
	},
	{
		name:        topicNameSoloSecret,
		description: "スペースの外からは見えない非公開トピックです。",
		visibility:  model.TopicVisibilityPrivate,
		memberRoles: []seedRole{roleOwner},
		assign:      func(topics *seededTopics, topic *seededTopic) { topics.soloSecret = topic },
	},
}

// seededTopic is one created topic. It carries its space id so that the
// generators that follow can keep space_id in the queries they write without
// having to reach back for the space.
//
// [Ja] seededTopic は作成したトピック 1 件。スペース ID を併せて持つことで、
// 後続の生成器がスペースを参照し直さずに、書くクエリへ space_id を入れられる。
type seededTopic struct {
	id      model.TopicID
	spaceID model.SpaceID
	name    string
}

// seededTopics holds the topics that later page generators write into. The
// topics of both spaces are named here: what an account that has not joined
// seed-solo sees is decided by the pages of its topics as much as by the
// topics themselves.
//
// [Ja] seededTopics は、後続のページ生成器がページを書き込むトピックを保持する。
// 両方のスペースのトピックがここに並ぶ。seed-solo に参加していないアカウントに
// 何が見えるかは、トピックそのものと同じくらいトピックのページによって
// 決まるため。
type seededTopics struct {
	handbook     *seededTopic
	notes        *seededTopic
	sandbox      *seededTopic
	privateNotes *seededTopic
	secret       *seededTopic
	soloNotes    *seededTopic
	soloSecret   *seededTopic
}

// generateTopics creates the topics of both spaces and joins the accounts to
// them.
//
// [Ja] generateTopics は両スペースのトピックを作成し、アカウントを参加させる。
func generateTopics(ctx context.Context, dbtx query.DBTX, out io.Writer, spaces *seededSpaces) (*seededTopics, error) {
	wikiSpecs, err := wikiTopicSpecs(spaces.wiki)
	if err != nil {
		return nil, err
	}

	bar := newProgress(out, "トピック", len(wikiSpecs)+len(soloTopicSpecs))
	defer bar.finish()

	topics := &seededTopics{}
	for _, group := range []struct {
		space *seededSpace
		specs []topicSpec
	}{
		{space: spaces.wiki, specs: wikiSpecs},
		{space: spaces.solo, specs: soloTopicSpecs},
	} {
		for i, spec := range group.specs {
			topic, err := createTopic(ctx, dbtx, group.space, spec, int32(i+1))
			if err != nil {
				return nil, err
			}
			if spec.assign != nil {
				spec.assign(topics, topic)
			}
			bar.advance()
		}
	}

	return topics, nil
}

// createTopic inserts one topic and joins to it the accounts the spec names.
//
// The rows are written here rather than through a repository because creating
// a topic is still handled by the Rails side, and the Go side has no Create to
// call.
//
// [Ja] createTopic はトピックを 1 つ INSERT し、仕様が指定するアカウントを参加させる。
//
// Repository ではなくここで行を書くのは、トピックの作成を担当しているのが今も
// Rails 側であり、Go 側に呼べる Create が無いため。
func createTopic(
	ctx context.Context,
	dbtx query.DBTX,
	space *seededSpace,
	spec topicSpec,
	number int32,
) (*seededTopic, error) {
	members := make([]*seededSpaceMember, 0, len(spec.memberRoles))
	for _, role := range spec.memberRoles {
		member := space.member(role)
		if member == nil {
			return nil, fmt.Errorf("トピック %s は役割 %s の参加を指定しているが、その役割はスペースに参加していない", spec.name, role)
		}
		members = append(members, member)
	}

	now := time.Now()

	var id string
	err := dbtx.QueryRowContext(
		ctx,
		`INSERT INTO topics (space_id, number, name, description, visibility, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7)
         RETURNING id`,
		string(space.id), number, spec.name, spec.description, int32(spec.visibility), now, now,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("トピック %s の作成に失敗: %w", spec.name, err)
	}

	topic := &seededTopic{id: model.TopicID(id), spaceID: space.id, name: spec.name}

	for _, member := range members {
		if err := addTopicMember(ctx, dbtx, topic, member, topicMemberScopes(spec.visibility, member)); err != nil {
			return nil, fmt.Errorf("トピック %s へのメンバー追加に失敗: %w", spec.name, err)
		}
	}

	return topic, nil
}

// topicMemberScopes returns the scopes a topic membership carries.
//
// Production creates topic memberships without scopes, because the only space
// membership it ever creates holds space:admin, which already expands to
// topic:read and so opens every topic. A member without space:admin gets no
// such expansion, and a private topic stays hidden from them even after they
// join it unless the membership itself carries topic:read. The seed grants it
// there, so that a private topic seen by a member who is not an admin is a
// state the screens can actually be opened in.
//
// [Ja] topicMemberScopes は、トピックメンバーシップが持つスコープを返す。
//
// 本番はトピックメンバーシップをスコープ無しで作る。本番が作るスペース
// メンバーシップは space:admin を持つものだけであり、それが既に topic:read へ
// 展開されて全トピックを開くため。space:admin を持たないメンバーにはその展開が
// 無く、非公開トピックは参加後も、メンバーシップ自身が topic:read を持たない限り
// 見えないままになる。シードはそこにこのスコープを与え、管理者でないメンバーから
// 見た非公開トピックを、画面として実際に開ける状態にする。
func topicMemberScopes(visibility model.TopicVisibility, member *seededSpaceMember) []model.Scope {
	if visibility == model.TopicVisibilityPublic {
		return nil
	}
	if model.HasScope(member.scopes, model.ScopeSpaceAdmin) {
		return nil
	}

	return []model.Scope{model.ScopeTopicRead}
}

// addTopicMember joins a space member to a topic with the given scopes.
//
// [Ja] addTopicMember は、指定のスコープでスペースメンバーをトピックに参加させる。
func addTopicMember(
	ctx context.Context,
	dbtx query.DBTX,
	topic *seededTopic,
	member *seededSpaceMember,
	scopes []model.Scope,
) error {
	now := time.Now()

	_, err := dbtx.ExecContext(
		ctx,
		`INSERT INTO topic_members (space_id, topic_id, space_member_id, scopes, joined_at, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		string(topic.spaceID), string(topic.id), string(member.id),
		pq.Array(scopeStrings(scopes)), now, now, now,
	)

	return err
}
