# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/atom", type: :request do
  it "ログインしていないとき、公開トピックのページ情報がAtomフィードに表示されること" do
    space = create(:space, :small)
    public_topic = create(:topic, :public, space:)
    private_topic = create(:topic, :private, space:)
    create(:page, :published, space:, topic: public_topic, title: "公開されているページ")
    create(:page, :published, space:, topic: private_topic, title: "公開されていないページ")

    get "/s/#{space.identifier}/atom"

    expect(response.status).to eq(200)
    expect(response.content_type).to eq("application/atom+xml; charset=utf-8")
    expect(response.body).to include("公開されているページ")
    expect(response.body).not_to include("公開されていないページ")
  end

  it "別のスペースにログインしているとき、公開トピックのページ情報がAtomフィードに表示されること" do
    space = create(:space, :small)
    public_topic = create(:topic, :public, space:)
    private_topic = create(:topic, :private, space:)
    create(:page, :published, space:, topic: public_topic, title: "公開されているページ")
    create(:page, :published, space:, topic: private_topic, title: "公開されていないページ")

    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    get "/s/#{space.identifier}/atom"

    expect(response.status).to eq(200)
    expect(response.content_type).to eq("application/atom+xml; charset=utf-8")
    expect(response.body).to include("公開されているページ")
    expect(response.body).not_to include("公開されていないページ")
  end

  it "同じスペースにログインしているとき、公開トピックのページ情報がAtomフィードに表示されること" do
    space = create(:space, :small)
    public_topic = create(:topic, :public, space:)
    private_topic = create(:topic, :private, space:)
    create(:page, :published, space:, topic: public_topic, title: "公開されているページ")
    create(:page, :published, space:, topic: private_topic, title: "公開されていないページ")
    user = create(:user, :with_password, space:)

    sign_in(user:)

    get "/s/#{space.identifier}/atom"

    expect(response.status).to eq(200)
    expect(response.content_type).to eq("application/atom+xml; charset=utf-8")
    expect(response.body).to include("公開されているページ")
    expect(response.body).not_to include("公開されていないページ")
  end
end
