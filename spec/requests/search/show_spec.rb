# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "GET /search", type: :request do
  it "検索ページが正常に表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    topic_record = create(:topic_record, space_record:)
    create(:space_member_record, user_record:, space_record:)
    sign_in(user_record:)

    get search_path
    expect(response).to have_http_status(:ok)
  end

  it "ログインしていない場合、ログインページにリダイレクトされること" do
    delete user_session_path
    get search_path
    expect(response).to redirect_to(sign_in_path)
  end

  it "検索キーワードがない場合、検索フォームのみが表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    topic_record = create(:topic_record, space_record:)
    create(:space_member_record, user_record:, space_record:)
    sign_in(user_record:)

    get search_path
    expect(response).to have_http_status(:ok)
    expect(response.body).to include("検索")
  end

  it "有効な検索キーワードがある場合、検索が実行されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    topic_record = create(:topic_record, space_record:)
    create(:space_member_record, user_record:, space_record:)
    sign_in(user_record:)

    create(:page_record,
      space_record:,
      topic_record:,
      title: "テストページ")

    get search_path, params: {q: "テスト"}

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("テストページ")
  end

  it "検索結果がない場合、適切なメッセージが表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    topic_record = create(:topic_record, space_record:)
    create(:space_member_record, user_record:, space_record:)
    sign_in(user_record:)

    get search_path, params: {q: "存在しないキーワード"}

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("検索結果が見つかりませんでした")
  end

  it "無効な検索キーワードの場合、エラーメッセージが表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    topic_record = create(:topic_record, space_record:)
    create(:space_member_record, user_record:, space_record:)
    sign_in(user_record:)

    get search_path, params: {q: "a"}

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("は2文字以上で入力してください")
  end

  it "他のユーザーのスペースのプライベートトピックのページは検索結果に含まれないこと" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    topic_record = create(:topic_record, space_record:)
    create(:space_member_record, user_record:, space_record:)
    sign_in(user_record:)

    other_user_record = create(:user_record, :with_password)
    other_space_record = create(:space_record)
    other_topic_record = create(:topic_record, space_record: other_space_record, visibility: TopicVisibility::Private.serialize)
    create(:space_member_record, user_record: other_user_record, space_record: other_space_record)
    create(:page_record,
      space_record: other_space_record,
      topic_record: other_topic_record,
      title: "他のユーザーのプライベートページ")

    get search_path, params: {q: "プライベートページ"}

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("検索結果が見つかりませんでした")
  end

  it "他のユーザーのスペースの公開トピックのページは検索結果に含まれること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    topic_record = create(:topic_record, space_record:)
    create(:space_member_record, user_record:, space_record:)
    sign_in(user_record:)

    other_user_record = create(:user_record, :with_password)
    other_space_record = create(:space_record)
    other_topic_record = create(:topic_record, space_record: other_space_record, visibility: TopicVisibility::Public.serialize)
    create(:space_member_record, user_record: other_user_record, space_record: other_space_record)
    create(:page_record,
      space_record: other_space_record,
      topic_record: other_topic_record,
      title: "他のユーザーの公開ページ")

    get search_path, params: {q: "公開ページ"}

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("他のユーザーの公開ページ")
  end

  describe "space:フィルター機能" do
    it "space:指定子で特定のスペース内のページを検索できること" do
      user_record = create(:user_record, :with_password)
      space1_record = create(:space_record, identifier: "space1")
      space2_record = create(:space_record, identifier: "space2")
      topic1_record = create(:topic_record, space_record: space1_record, visibility: TopicVisibility::Private.serialize)
      topic2_record = create(:topic_record, space_record: space2_record, visibility: TopicVisibility::Private.serialize)
      create(:space_member_record, user_record:, space_record: space1_record)
      create(:space_member_record, user_record:, space_record: space2_record)
      sign_in(user_record:)

      create(:page_record,
        space_record: space1_record,
        topic_record: topic1_record,
        title: "Space1のテストページ")
      create(:page_record,
        space_record: space2_record,
        topic_record: topic2_record,
        title: "Space2のテストページ")

      get search_path, params: {q: "space:space1 テスト"}

      expect(response).to have_http_status(:ok)
      expect(response.body).to include("Space1のテストページ")
      expect(response.body).not_to include("Space2のテストページ")
    end

    it "複数のspace:指定子で複数のスペース内のページを検索できること" do
      user_record = create(:user_record, :with_password)
      space1_record = create(:space_record, identifier: "space1")
      space2_record = create(:space_record, identifier: "space2")
      topic1_record = create(:topic_record, space_record: space1_record, visibility: TopicVisibility::Private.serialize)
      topic2_record = create(:topic_record, space_record: space2_record, visibility: TopicVisibility::Private.serialize)
      create(:space_member_record, user_record:, space_record: space1_record)
      create(:space_member_record, user_record:, space_record: space2_record)
      sign_in(user_record:)

      create(:page_record,
        space_record: space1_record,
        topic_record: topic1_record,
        title: "Space1のテストページ")
      create(:page_record,
        space_record: space2_record,
        topic_record: topic2_record,
        title: "Space2のテストページ")

      get search_path, params: {q: "space:space1 space:space2 テスト"}

      expect(response).to have_http_status(:ok)
      expect(response.body).to include("Space1のテストページ")
      expect(response.body).to include("Space2のテストページ")
    end

    it "space:指定子のみでキーワードがない場合、そのスペースの全ページが表示されること" do
      user_record = create(:user_record, :with_password)
      space1_record = create(:space_record, identifier: "space1")
      space2_record = create(:space_record, identifier: "space2")
      topic1_record = create(:topic_record, space_record: space1_record, visibility: TopicVisibility::Private.serialize)
      topic2_record = create(:topic_record, space_record: space2_record, visibility: TopicVisibility::Private.serialize)
      create(:space_member_record, user_record:, space_record: space1_record)
      create(:space_member_record, user_record:, space_record: space2_record)
      sign_in(user_record:)

      create(:page_record,
        space_record: space1_record,
        topic_record: topic1_record,
        title: "Space1のテストページ")
      create(:page_record,
        space_record: space2_record,
        topic_record: topic2_record,
        title: "Space2のテストページ")

      get search_path, params: {q: "space:space1"}

      expect(response).to have_http_status(:ok)
      expect(response.body).to include("Space1のテストページ")
      expect(response.body).not_to include("Space2のテストページ")
    end

    it "存在しないスペース識別子の場合、検索結果が空になること" do
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record)
      topic_record = create(:topic_record, space_record:)
      create(:space_member_record, user_record:, space_record:)
      sign_in(user_record:)

      get search_path, params: {q: "space:nonexistent"}

      expect(response).to have_http_status(:ok)
      expect(response.body).to include("検索結果が見つかりませんでした")
    end

    it "参加していないスペースのプライベートトピックは検索できないこと" do
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record)
      topic_record = create(:topic_record, space_record:)
      create(:space_member_record, user_record:, space_record:)
      sign_in(user_record:)

      other_space_record = create(:space_record, identifier: "other-space")
      other_topic_record = create(:topic_record, space_record: other_space_record, visibility: TopicVisibility::Private.serialize)
      create(:page_record,
        space_record: other_space_record,
        topic_record: other_topic_record,
        title: "参加していないスペースのプライベートページ")

      get search_path, params: {q: "space:other-space"}

      expect(response).to have_http_status(:ok)
      expect(response.body).to include("検索結果が見つかりませんでした")
    end

    it "参加していないスペースの公開トピックは検索できること" do
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record)
      topic_record = create(:topic_record, space_record:)
      create(:space_member_record, user_record:, space_record:)
      sign_in(user_record:)

      other_space_record = create(:space_record, identifier: "other-space")
      other_topic_record = create(:topic_record, space_record: other_space_record, visibility: TopicVisibility::Public.serialize)
      create(:page_record,
        space_record: other_space_record,
        topic_record: other_topic_record,
        title: "参加していないスペースの公開ページ")

      get search_path, params: {q: "space:other-space 公開ページ"}

      expect(response).to have_http_status(:ok)
      expect(response.body).to include("参加していないスペースの公開ページ")
    end
  end
end
