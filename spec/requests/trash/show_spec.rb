# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/trash", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)

    get "/s/#{space.identifier}/trash"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースにログインしているとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    get "/s/#{space.identifier}/trash"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしているとき、ゴミ箱ページが表示されること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)
    topic = create(:topic, space:)
    create(:page, :trashed, space:, topic:, title: "削除されたページ")

    sign_in(user:)

    get "/s/#{space.identifier}/trash"

    expect(response.status).to eq(200)
    expect(response.body).to include("ゴミ箱に入れたページを表示しています。")
    expect(response.body).to include("削除されたページ")
  end
end
