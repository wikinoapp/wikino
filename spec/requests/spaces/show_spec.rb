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

    SoftDestroyTopicService.new.call(topic_record:)

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
end
