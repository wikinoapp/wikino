# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier", type: :request do
  it "トピックが削除されているとき、そのトピックに投稿されたページは表示されないこと" do
    space_record = create(:space_record)
    topic_record = create(:topic_record, :public, space_record:)
    create(:page_record, :published, space_record:, topic_record:, title: "テストページ")

    get "/s/#{space_record.identifier}"

    expect(response.status).to eq(200)
    expect(response.body).to include("テストページ")

    Topics::SoftDestroyService.new.call(topic_record:)

    get "/s/#{space_record.identifier}"

    expect(response.status).to eq(200)
    expect(response.body).not_to include("テストページ")
  end

  it "ページがゴミ箱にあるとき、そのページは表示されないこと" do
    space_record = create(:space_record)
    topic_record = create(:topic_record, :public, space_record:)
    create(:page_record, :published, :trashed, space_record:, topic_record:, title: "テストページ")

    get "/s/#{space_record.identifier}"

    expect(response.status).to eq(200)
    expect(response.body).not_to include("テストページ")
  end

  it "ログインしていないとき、公開トピックのページが表示されること" do
    space = create(:space_record, :small)
    public_topic = create(:topic_record, :public, space_record: space)
    private_topic = create(:topic_record, :private, space_record: space)
    create(:page_record, :published, space_record: space, topic_record: public_topic, title: "公開されているページ")
    create(:page_record, :published, space_record: space, topic_record: private_topic, title: "公開されていないページ")

    get "/s/#{space.identifier}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    expect(response.body).not_to include("公開されていないページ")
  end

  it "別のスペースに参加しているとき、公開トピックのページが表示されること" do
    user = create(:user_record, :with_password)

    space = create(:space_record, :small)
    public_topic = create(:topic_record, :public, space_record: space)
    private_topic = create(:topic_record, :private, space_record: space)
    create(:page_record, :published, space_record: space, topic_record: public_topic, title: "公開されているページ")
    create(:page_record, :published, space_record: space, topic_record: private_topic, title: "公開されていないページ")

    other_space = create(:space_record)
    create(:space_member_record, space_record: other_space, user_record: user)

    sign_in(user_record: user)

    get "/s/#{space.identifier}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    expect(response.body).not_to include("公開されていないページ")
  end

  it "スペースに参加しているとき、公開/非公開トピックのページが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    space_member = create(:space_member_record, space_record: space, user_record: user)

    public_topic = create(:topic_record, :public, space_record: space)
    private_topic = create(:topic_record, :private, space_record: space)
    not_joined_public_topic = create(:topic_record, :public, space_record: space)
    not_joined_private_topic = create(:topic_record, :private, space_record: space)
    create(:topic_member_record, space_record: space, topic_record: public_topic, space_member_record: space_member)
    create(:topic_member_record, space_record: space, topic_record: private_topic, space_member_record: space_member)
    create(:page_record, :published, space_record: space, topic_record: public_topic, title: "公開されているページ")
    create(:page_record, :published, space_record: space, topic_record: private_topic, title: "公開されていないページ")
    create(:page_record, :published, space_record: space, topic_record: not_joined_public_topic, title: "参加していない公開トピックのページ")
    create(:page_record, :published, space_record: space, topic_record: not_joined_private_topic, title: "参加していない非公開トピックのページ")

    sign_in(user_record: user)

    get "/s/#{space.identifier}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    expect(response.body).to include("公開されていないページ")
    expect(response.body).to include("参加していない公開トピックのページ")
    expect(response.body).to include("参加していない非公開トピックのページ")
  end

  describe "トピックリンク表示機能" do
    it "スペースメンバーの場合、参加しているトピックが表示されること" do
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record)
      space_member_record = create(:space_member_record, :member, user_record:, space_record:)

      # トピック作成
      topic_record1 = create(:topic_record,
        space_record:,
        name: "参加しているトピック")
      topic_record2 = create(:topic_record,
        space_record:,
        name: "参加していないトピック")

      # topic_record1にのみ参加
      create(:topic_member_record,
        space_record:,
        topic_record: topic_record1,
        space_member_record:)

      # サインイン
      sign_in(user_record:)

      # リクエスト実行
      get "/s/#{space_record.identifier}"

      # 検証
      expect(response.status).to eq(200)
      expect(response.body).to include("参加しているトピック")
      expect(response.body).not_to include("参加していないトピック")
    end

    it "ゲストの場合、公開トピックのみが表示されること" do
      space_record = create(:space_record)

      # トピック作成
      create(:topic_record,
        space_record:,
        name: "公開トピック",
        visibility: TopicVisibility::Public.serialize)
      create(:topic_record,
        space_record:,
        name: "非公開トピック",
        visibility: TopicVisibility::Private.serialize)

      # リクエスト実行（未ログイン）
      get "/s/#{space_record.identifier}"

      # 検証
      expect(response.status).to eq(200)
      expect(response.body).to include("公開トピック")
      expect(response.body).not_to include("非公開トピック")
    end

    it "スペースメンバーでページ作成権限がある場合、ページ作成ボタンが表示されること" do
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record)
      space_member_record = create(:space_member_record, :member, user_record:, space_record:)
      topic_record = create(:topic_record, space_record:)

      # トピックメンバーとして追加（管理者権限）
      create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      # サインイン
      sign_in(user_record:)

      # リクエスト実行
      get "/s/#{space_record.identifier}"

      # 検証
      expect(response.status).to eq(200)
      # ページ作成リンクが存在することを確認
      expect(response.body).to include("/s/#{space_record.identifier}/topics/#{topic_record.number}/pages/new")
    end

    it "トピックの説明が存在する場合、表示されること" do
      space_record = create(:space_record)
      create(:topic_record,
        space_record:,
        name: "説明付きトピック",
        description: "これはトピックの説明です",
        visibility: TopicVisibility::Public.serialize)

      # リクエスト実行（未ログイン）
      get "/s/#{space_record.identifier}"

      # 検証
      expect(response.status).to eq(200)
      expect(response.body).to include("説明付きトピック")
      expect(response.body).to include("これはトピックの説明です")
    end
  end
end
