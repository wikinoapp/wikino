# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier", type: :request do
  it "ログインしていないとき、公開トピックのページが表示されること" do
    space = create(:space, :small)
    public_topic = create(:topic, :public, space:)
    private_topic = create(:topic, :private, space:)
    create(:page, :published, space:, topic: public_topic, title: "公開されているページ")
    create(:page, :published, space:, topic: private_topic, title: "公開されていないページ")

    get "/s/#{space.identifier}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    expect(response.body).not_to include("公開されていないページ")
  end

  it "別のスペースに参加しているとき、公開トピックのページが表示されること" do
    user = create(:user, :with_password)

    space = create(:space, :small)
    public_topic = create(:topic, :public, space:)
    private_topic = create(:topic, :private, space:)
    create(:page, :published, space:, topic: public_topic, title: "公開されているページ")
    create(:page, :published, space:, topic: private_topic, title: "公開されていないページ")

    other_space = create(:space)
    create(:space_member, space: other_space, user:)

    sign_in(user:)

    get "/s/#{space.identifier}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    expect(response.body).not_to include("公開されていないページ")
  end

  it "スペースに参加しているとき、自分が参加している公開/非公開トピックのページが表示されること" do
    user = create(:user, :with_password)
    space = create(:space, :small)
    space_member = create(:space_member, space:, user:)

    public_topic = create(:topic, :public, space:)
    private_topic = create(:topic, :private, space:)
    not_joined_public_topic = create(:topic, :public, space:)
    not_joined_private_topic = create(:topic, :private, space:)
    create(:topic_membership, space:, topic: public_topic, member: space_member)
    create(:topic_membership, space:, topic: private_topic, member: space_member)
    create(:page, :published, space:, topic: public_topic, title: "公開されているページ")
    create(:page, :published, space:, topic: private_topic, title: "公開されていないページ")
    create(:page, :published, space:, topic: not_joined_public_topic, title: "参加していない公開トピックのページ")
    create(:page, :published, space:, topic: not_joined_private_topic, title: "参加していない非公開トピックのページ")

    sign_in(user:)

    get "/s/#{space.identifier}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    expect(response.body).to include("公開されていないページ")
    expect(response.body).not_to include("参加していない公開トピックのページ")
    expect(response.body).not_to include("参加していない非公開トピックのページ")
  end
end
