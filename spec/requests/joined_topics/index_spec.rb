# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/joined_topics", type: :request do
  it "ログインしていないとき、公開トピックが表示されること" do
    space = create(:space, :small)
    create(:topic, :public, space:, name: "公開トピック")
    create(:topic, :private, space:, name: "非公開トピック")

    get "/s/#{space.identifier}/joined_topics"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開トピック")
    expect(response.body).not_to include("非公開トピック")
  end

  it "別のスペースにログインしているとき、アクセスしたスペースの公開トピックが表示されること" do
    space = create(:space, :small)
    create(:topic, :public, space:, name: "アクセスしたスペースの公開トピック")
    create(:topic, :private, space:, name: "アクセスしたスペースの非公開トピック")

    other_space = create(:space)
    create(:topic, :public, space: other_space, name: "別のスペースの公開トピック")
    create(:topic, :private, space: other_space, name: "別のスペースの非公開トピック")
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    get "/s/#{space.identifier}/joined_topics"

    expect(response.status).to eq(200)
    expect(response.body).to include("アクセスしたスペースの公開トピック")
    expect(response.body).not_to include("アクセスしたスペースの非公開トピック")
    expect(response.body).not_to include("別のスペースの公開トピック")
    expect(response.body).not_to include("別のスペースの非公開トピック")
  end

  it "同じスペースにログインしているとき、自分が参加している公開/非公開トピックが表示されること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)
    public_topic = create(:topic, :public, space:, name: "参加している公開トピック")
    private_topic = create(:topic, :private, space:, name: "参加している非公開トピック")
    create(:topic, :public, space:, name: "参加していない公開トピック")
    create(:topic, :private, space:, name: "参加していない非公開トピック")
    create(:topic_membership, space:, topic: public_topic, member: user)
    create(:topic_membership, space:, topic: private_topic, member: user)

    sign_in(user:)

    get "/s/#{space.identifier}/joined_topics"

    expect(response.status).to eq(200)
    expect(response.body).to include("参加している公開トピック")
    expect(response.body).to include("参加している非公開トピック")
    expect(response.body).not_to include("参加していない公開トピック")
    expect(response.body).not_to include("参加していない非公開トピック")
  end
end
