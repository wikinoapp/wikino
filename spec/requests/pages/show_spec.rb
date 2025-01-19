# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/pages/:page_number", type: :request do
  it "ログインしていない & 公開トピックのページのとき、ページが表示されること" do
    space = create(:space, :small)
    public_topic = create(:topic, :public, space:)
    page = create(:page, space:, topic: public_topic, title: "公開されているページ")

    get "/s/#{space.identifier}/pages/#{page.number}"
    page = Capybara.string(response.body)

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
    expect(page).to have_no_content("ゴミ箱に入れる")
  end

  it "ログインしていない & 非公開トピックのページのとき、404を返すこと" do
    space = create(:space, :small)
    private_topic = create(:topic, :private, space:)
    page = create(:page, space:, topic: private_topic)

    get "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(404)
  end

  it "別のスペースにログインしている & 公開トピックのページのとき、ページが表示されること" do
    space = create(:space, :small)
    public_topic = create(:topic, :public, space:)
    page = create(:page, space:, topic: public_topic, title: "公開されているページ")

    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    get "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
  end

  it "別のスペースにログインしている & 非公開トピックのページのとき、404を返すこと" do
    space = create(:space, :small)
    private_topic = create(:topic, :private, space:)
    page = create(:page, space:, topic: private_topic)

    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    get "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(404)
  end

  it "ログインしている & 参加している公開トピックのページのとき、ページが表示されること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)
    topic = create(:topic, :public, space:)
    create(:topic_membership, space:, topic:, member: user)
    page = create(:page, space:, topic:, title: "公開されているページ")

    sign_in(user:)

    get "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
  end

  it "ログインしている & 参加している非公開トピックのページのとき、ページが表示されること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)
    topic = create(:topic, :private, space:)
    create(:topic_membership, space:, topic:, member: user)
    page = create(:page, space:, topic:, title: "公開されていないページ")

    sign_in(user:)

    get "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されていないページ")
  end

  it "ログインしている & 参加していない公開トピックのページのとき、ページが表示されること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)
    topic = create(:topic, :public, space:)
    page = create(:page, space:, topic:, title: "公開されているページ")

    sign_in(user:)

    get "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されているページ")
  end

  it "ログインしている & 参加していない非公開トピックのページのとき、ページが表示されること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)
    topic = create(:topic, :private, space:)
    page = create(:page, space:, topic:, title: "公開されていないページ")

    sign_in(user:)

    get "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(200)
    expect(response.body).to include("公開されていないページ")
  end
end
