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

// トピック名を定数にしているのは、後続の生成器が書き込み先のトピックを名前で
// 選ぶことと、ブラウザ確認がこの名前を辿ることによる。
const (
	topicNameHandbook     = "ハンドブック"
	topicNameNotes        = "ノート"
	topicNameSandbox      = "サンドボックス"
	topicNamePrivateNotes = "非公開ノート"
	topicNameSecret       = "シークレット"
	topicNameSoloNotes    = "個人ノート"
	topicNameSoloSecret   = "個人シークレット"
	topicNameLongJapanese = "折り返しの確認用に名前を最大文字数まで伸ばした公開のトピック"
	topicNameLongASCII    = "UnbreakableLongTopicNameSample"
	topicNameDemoMemo     = "Memo"
	topicNameExport       = "エクスポート"
	topicNameExportSymbol = "エクスポート*記号"
)

// topicSpecは作成するトピック1件の内容。仕様を順序付きの一覧として持つのは、
// スペース内で一意であり一覧の並び順を決めるトピック番号を、手書きせずに位置から
// 採番するため。
type topicSpec struct {
	name        string
	description string
	visibility  model.TopicVisibility
	// memberRolesはトピックに参加させるアカウント。スペースから導かずここで
	// 名指しするのは、1つの役割だけが読めるトピックが、後からスペースへ足した
	// アカウントに開いてしまわないようにするため。
	memberRoles []seedRole
	// assignは、後続の生成器がページを書き込むトピックについて、作成した
	// トピックを結果へ格納する。それ以外ではnil。
	assign func(topics *seededTopics, topic *seededTopic)
}

// wikiTopicSpecsはseed-wikiのトピックを返す。全体で、トピック画面が取りうる
// 状態を網羅する。公開と非公開、複数のアカウントが参加しているものと1つだけが
// 参加しているもの、ページが大量にあるものと無いもの。
//
// 仕様をパッケージ変数の一覧ではなくスペースから組み立てるのは、非公開トピックの
// 説明文が、そこに参加しているアカウントを名指しするため。そのアカウントが何と
// 呼ばれるのかは名簿に書かれている。名前をメンバーシップから読むことが、トピックの
// 説明文と、その中にいるアカウントとが同じことを述べ続ける理由になる。
//
// 名前は説明文へそのまま入れる。ページ本文へ入れる場合とは違う。説明文はtemplを
// 通ってテキストとして画面へ出て、templがエスケープする。Markdownとして読む経路が
// 無いため、名前に書かれた記法が画面で効くこともない。本文と同じくmarkdownPlainText
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
		// エクスポートの2つを末尾に置いているのは意図的である。トピック番号は
		// この一覧での位置から採番され、ブラウザ確認はURLの番号で画面を指すため、
		// これらより上に仕様を挟むと、そこまでに記録した画面がすべてずれる。
		{
			name:        topicNameExport,
			description: "エクスポートした ZIP で、ページタイトルがどうファイル名になるかを確認するためのトピックです。記号の置き換え・Windows の予約名・大文字小文字だけが違う名前の衝突・Wiki リンクの書き換えを見ます。",
			visibility:  model.TopicVisibilityPublic,
			memberRoles: []seedRole{roleOwner, roleCollaborator},
			assign:      func(topics *seededTopics, topic *seededTopic) { topics.export = topic },
		},
		{
			name: topicNameExportSymbol,
			// このトピックは名前自体が確認の対象になる。名前が持つ半角の "*" が、
			// アーカイブがトピックへ与えるディレクトリ名で "＊" に変わる部分である。
			// それを説明文に書くのは、この名前が一覧で他と並んだときに目立つ一方、
			// なぜそう綴られているのかを画面の他のどこも説明しないためである。
			description: "名前の半角記号が ZIP の中でディレクトリ名「エクスポート＊記号」に変わることを、名前自体で確認するためのトピックです。frontmatter を持つ本文の扱いを見るページを置きます。",
			visibility:  model.TopicVisibilityPublic,
			memberRoles: []seedRole{roleOwner, roleCollaborator},
			assign:      func(topics *seededTopics, topic *seededTopic) { topics.exportSymbol = topic },
		},
	}, nil
}

// soloTopicSpecsはseed-soloのトピック。このスペースに参加しているのは
// roleOwnerだけであるため、この2つでスペースを外から見た状態を確認できる。
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

// longNameTopicSpecsはseed-long-nameのトピック。どちらの名前もトピック名が
// 取りうる最大の長さで、違いはテキストを分割できるかどうかにある。1つ目は日本語で、
// 任意の文字位置で折り返せる。2つ目は途切れない半角英字の連なりで、折り返せる場所が
// 無い。1つ目に耐えるレイアウトでも2つ目では崩れうるため、両方を置いている。
//
// 名前はどちらも30文字で、トピックの作成を今も担当しているRails側の
// Topic::NAME_MAX_LENGTHが許す最大の長さ。
//
// どちらもassignしない。これらのトピックへページを書き込む生成器は無い。ここで
// 示すのは自身の名前の描かれ方であり、それはトピックがページを持つかどうかに
// 関わらずトピック一覧とトピック画面に出る。
var longNameTopicSpecs = []topicSpec{
	{
		name:        topicNameLongJapanese,
		description: "名前を最大文字数まで伸ばした公開トピックです。",
		visibility:  model.TopicVisibilityPublic,
		memberRoles: []seedRole{roleOwner},
	},
	{
		name:        topicNameLongASCII,
		description: "名前を、折り返せない半角英字だけで最大文字数まで伸ばした公開トピックです。",
		visibility:  model.TopicVisibilityPublic,
		memberRoles: []seedRole{roleOwner},
	},
}

// demoTopicSpecsはデモスペースのトピック。1つだけで、デモページはすべて
// そこへ書き込む。本文どうしはタイトルだけでリンクし合っており、トピック名を
// 伴わないWikiリンクは、それが書かれたトピックの中でしか解決しない。そのため
// ジャンルでページを分けると、ジャンルを跨ぐリンクがすべて存在しないページを
// 指すことになる。
//
// 非公開にしているのは、このスペースが個人が自分のために持つメモ置き場であり、
// その種のトピックはそう持たれるため。それで失うものは無い。参加する唯一の
// アカウントはspace:adminを持つため非公開トピックも開けるうえ、このスペースを
// 外から見たスクリーンショットを撮ることも無い。
var demoTopicSpecs = []topicSpec{
	{
		name:        topicNameDemoMemo,
		description: "日々の覚え書きです。行った場所や読んだもの、作ったものを書きためています。",
		visibility:  model.TopicVisibilityPrivate,
		memberRoles: []seedRole{roleOwner},
		assign:      func(topics *seededTopics, topic *seededTopic) { topics.demoMemo = topic },
	},
}

// seededTopicは作成したトピック1件。スペースIDを併せて持つことで、
// 後続の生成器がスペースを参照し直さずに、書くクエリへspace_idを入れられる。
type seededTopic struct {
	id      model.TopicID
	spaceID model.SpaceID
	name    string
}

// seededTopicsは、後続のページ生成器がページを書き込むトピックを保持する。
// seed-wikiとseed-soloのトピックがここに並ぶ。seed-soloに参加していない
// アカウントに何が見えるかは、トピックそのものと同じくらいトピックのページによって
// 決まるため。デモのトピックが並ぶ理由はもっと単純で、デモページがすべてそこへ
// 入るからである。エクスポートの2つも同じで、エクスポートが何を変換するのかを
// 示すページがそこへ書き込まれる。seed-long-nameのトピックは、そこへ書き込む
// 生成器が無いため並ばない。
type seededTopics struct {
	handbook     *seededTopic
	notes        *seededTopic
	sandbox      *seededTopic
	privateNotes *seededTopic
	secret       *seededTopic
	soloNotes    *seededTopic
	soloSecret   *seededTopic
	demoMemo     *seededTopic
	export       *seededTopic
	exportSymbol *seededTopic
}

// generateTopicsは各スペースのトピックを作成し、アカウントを参加させる。
func generateTopics(ctx context.Context, dbtx query.DBTX, out io.Writer, spaces *seededSpaces) (*seededTopics, error) {
	wikiSpecs, err := wikiTopicSpecs(spaces.wiki)
	if err != nil {
		return nil, err
	}

	bar := newProgress(out, "トピック", len(wikiSpecs)+len(soloTopicSpecs)+len(longNameTopicSpecs)+len(demoTopicSpecs))
	defer bar.finish()

	topics := &seededTopics{}
	for _, group := range []struct {
		space *seededSpace
		specs []topicSpec
	}{
		{space: spaces.wiki, specs: wikiSpecs},
		{space: spaces.solo, specs: soloTopicSpecs},
		{space: spaces.longName, specs: longNameTopicSpecs},
		{space: spaces.demo, specs: demoTopicSpecs},
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

// createTopicはトピックを1つINSERTし、仕様が指定するアカウントを参加させる。
//
// Repositoryではなくここで行を書くのは、トピックの作成を担当しているのが今も
// Rails側であり、Go側に呼べるCreateが無いため。
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
			return nil, fmt.Errorf("トピック %sは役割 %sの参加を指定しているが、その役割はスペースに参加していない", spec.name, role)
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
		return nil, fmt.Errorf("トピック %sの作成に失敗: %w", spec.name, err)
	}

	topic := &seededTopic{id: model.TopicID(id), spaceID: space.id, name: spec.name}

	for _, member := range members {
		if err := addTopicMember(ctx, dbtx, topic, member, topicMemberScopes(spec.visibility, member)); err != nil {
			return nil, fmt.Errorf("トピック %sへのメンバー追加に失敗: %w", spec.name, err)
		}
	}

	return topic, nil
}

// topicMemberScopesは、トピックメンバーシップが持つスコープを返す。
//
// 本番はトピックメンバーシップをスコープ無しで作る。本番が作るスペース
// メンバーシップはspace:adminを持つものだけであり、それが既にtopic:readへ
// 展開されて全トピックを開くため。space:adminを持たないメンバーにはその展開が
// 無く、非公開トピックは参加後も、メンバーシップ自身がtopic:readを持たない限り
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

// addTopicMemberは、指定のスコープでスペースメンバーをトピックに参加させる。
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
