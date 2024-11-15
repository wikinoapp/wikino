# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/topics/:topic_number", type: :request do
  it "ログインしていない & 公開トピックのとき、ページが表示されること" do
    space = create(:space, :small)
    topic = create(:topic, :public, space:, name: "公開されているトピック")

    get "/s/#{space.identifier}/topics/#{topic.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているトピック")
  end

  it "ログインしていない & 非公開トピックのとき、404を返すこと" do
    space = create(:space, :small)
    topic = create(:topic, :private, space:)

    get "/s/#{space.identifier}/topics/#{topic.number}"

    expect(response.status).to eq(404)
  end

  it "別のスペースにログインしている & 公開トピックのとき、ページが表示されること" do
    space = create(:space, :small)
    topic = create(:topic, :public, space:, name: "公開されているトピック")

    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    get "/s/#{space.identifier}/topics/#{topic.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているトピック")
  end

  it "別のスペースにログインしている & 非公開トピックのとき、404を返すこと" do
    space = create(:space, :small)
    topic = create(:topic, :private, space:)

    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    get "/s/#{space.identifier}/topics/#{topic.number}"

    expect(response.status).to eq(404)
  end

  it "ログインしている & 参加している公開トピックのとき、ページが表示されること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)
    topic = create(:topic, :public, space:, name: "公開されているトピック")
    create(:topic_membership, space:, topic:, member: user)

    sign_in(user:)

    get "/s/#{space.identifier}/topics/#{topic.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているトピック")
  end

  it "ログインしている & 参加している非公開トピックのとき、ページが表示されること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)
    topic = create(:topic, :private, space:, name: "公開されていないトピック")
    create(:topic_membership, space:, topic:, member: user)

    sign_in(user:)

    get "/s/#{space.identifier}/topics/#{topic.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されていないトピック")
  end

  it "ログインしている & 参加していない公開トピックのとき、ページが表示されること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)
    topic = create(:topic, :public, space:, name: "公開されているトピック")

    sign_in(user:)

    get "/s/#{space.identifier}/topics/#{topic.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているトピック")
  end

  it "ログインしている & 参加していない非公開トピックのとき、ページが表示されること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)
    topic = create(:topic, :private, space:, name: "公開されていないトピック")

    sign_in(user:)

    get "/s/#{space.identifier}/topics/#{topic.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されていないトピック")
  end
end
