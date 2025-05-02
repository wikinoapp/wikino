# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/atom", type: :request do
  it "トピックが削除されているとき、そのトピックに投稿されたページはAtomフィードに表示されないこと" do
    space_record = create(:space_record)
    topic_record = create(:topic_record, :public, space_record:)
    create(:page_record, :published, space_record:, topic_record:, title: "テストページ")

    get "/s/#{space_record.identifier}/atom"

    expect(response.status).to eq(200)
    expect(response.body).to include("テストページ")

    SoftDestroyTopicService.new.call(topic_record:)

    get "/s/#{space_record.identifier}/atom"

    expect(response.status).to eq(200)
    expect(response.body).not_to include("テストページ")
  end

  it "ページがゴミ箱にあるとき、そのページはAtomフィードに表示されないこと" do
    space_record = create(:space_record)
    topic_record = create(:topic_record, :public, space_record:)
    create(:page_record, :published, :trashed, space_record:, topic_record:, title: "テストページ")

    get "/s/#{space_record.identifier}/atom"

    expect(response.status).to eq(200)
    expect(response.body).not_to include("テストページ")
  end

  it "ログインしていないとき、公開トピックのページ情報がAtomフィードに表示されること" do
    space = create(:space_record, :small)
    public_topic = create(:topic_record, :public, space_record: space)
    private_topic = create(:topic_record, :private, space_record: space)
    create(:page_record, :published, space_record: space, topic_record: public_topic, title: "公開されているページ")
    create(:page_record, :published, space_record: space, topic_record: private_topic, title: "公開されていないページ")

    get "/s/#{space.identifier}/atom"

    expect(response.status).to eq(200)
    expect(response.content_type).to eq("application/atom+xml; charset=utf-8")
    expect(response.body).to include("公開されているページ")
    expect(response.body).not_to include("公開されていないページ")
  end

  it "別のスペースに参加しているとき、公開トピックのページ情報がAtomフィードに表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    public_topic = create(:topic_record, :public, space_record: space)
    private_topic = create(:topic_record, :private, space_record: space)
    create(:page_record, :published, space_record: space, topic_record: public_topic, title: "公開されているページ")
    create(:page_record, :published, space_record: space, topic_record: private_topic, title: "公開されていないページ")

    other_space = create(:space_record)
    create(:space_member_record, user_record: user, space_record: other_space)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/atom"

    expect(response.status).to eq(200)
    expect(response.content_type).to eq("application/atom+xml; charset=utf-8")
    expect(response.body).to include("公開されているページ")
    expect(response.body).not_to include("公開されていないページ")
  end

  it "スペースに参加しているとき、公開トピックのページ情報がAtomフィードに表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    create(:space_member_record, user_record: user, space_record: space)

    public_topic = create(:topic_record, :public, space_record: space)
    private_topic = create(:topic_record, :private, space_record: space)
    create(:page_record, :published, space_record: space, topic_record: public_topic, title: "公開されているページ")
    create(:page_record, :published, space_record: space, topic_record: private_topic, title: "公開されていないページ")

    sign_in(user_record: user)

    get "/s/#{space.identifier}/atom"

    expect(response.status).to eq(200)
    expect(response.content_type).to eq("application/atom+xml; charset=utf-8")
    expect(response.body).to include("公開されているページ")
    expect(response.body).not_to include("公開されていないページ")
  end
end
