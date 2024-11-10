# typed: false
# frozen_string_literal: true

RSpec.describe "GET /", type: :request do
  it "ログインしていないとき、公開トピックのページが表示されること" do
    space = create(:space, :small)
    public_topic = create(:topic, :public, space:)
    private_topic = create(:topic, :private, space:)
    create(:page, :published, space:, topic: public_topic, title: "公開されているページ")
    create(:page, :published, space:, topic: private_topic, title: "公開されていないページ")

    host! space.host_name
    get "/"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    expect(response.body).not_to include("公開されていないページ")
  end

  it "別のスペースにログインしているとき、公開トピックのページが表示されること" do
    space = create(:space, :small)
    public_topic = create(:topic, :public, space:)
    private_topic = create(:topic, :private, space:)
    create(:page, :published, space:, topic: public_topic, title: "公開されているページ")
    create(:page, :published, space:, topic: private_topic, title: "公開されていないページ")

    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    host! space.host_name
    get "/"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    expect(response.body).not_to include("公開されていないページ")
  end

  it "同じスペースにログインしているとき、自分が参加している公開/非公開トピックのページが表示されること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)
    public_topic = create(:topic, :public, space:)
    private_topic = create(:topic, :private, space:)
    not_joined_topic = create(:topic, space:)
    create(:topic_membership, space:, topic: public_topic, member: user)
    create(:topic_membership, space:, topic: private_topic, member: user)
    create(:page, :published, space:, topic: public_topic, title: "公開されているページ")
    create(:page, :published, space:, topic: private_topic, title: "公開されていないページ")
    create(:page, :published, space:, topic: not_joined_topic, title: "参加していないトピックのページ")

    sign_in(user:)

    host! space.host_name
    get "/"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    expect(response.body).to include("公開されていないページ")
    expect(response.body).not_to include("参加していないトピックのページ")
  end
end
