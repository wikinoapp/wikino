# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/trash", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)

    get "/s/#{space.identifier}/trash"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "スペースに参加していないとき、404を返すこと" do
    space = create(:space, :small)
    other_space = create(:space_record)
    user = create(:user_record, :with_password)
    create(:space_member, :owner, space: other_space, user:)

    sign_in(user:)

    get "/s/#{space.identifier}/trash"

    expect(response.status).to eq(404)
  end

  it "スペースに参加しているとき、ゴミ箱ページが表示されること" do
    space = create(:space, :small)
    user = create(:user_record, :with_password)
    create(:space_member, :owner, space:, user:)
    topic = create(:topic, space:)
    create(:page, :trashed, space:, topic:, title: "削除されたページ")

    sign_in(user:)

    get "/s/#{space.identifier}/trash"

    expect(response.status).to eq(200)
    expect(response.body).to include("ゴミ箱に入れたページを表示しています。")
    expect(response.body).to include("削除されたページ")
  end
end
