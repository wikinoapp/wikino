# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/switchable_accounts", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)

    get "/s/#{space.identifier}/switchable_accounts"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースにログインしているとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    get "/s/#{space.identifier}/switchable_accounts"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしているとき、切り替え可能なアカウントの一覧が表示されること" do
    space_1 = create(:space, :small, name: "スペース1")
    space_2 = create(:space, :small, name: "スペース2")
    user_1 = create(:user, :with_password, space: space_1)
    user_2 = create(:user, :with_password, space: space_2)

    sign_in(user: user_1)
    sign_in(user: user_2)

    get "/s/#{space_1.identifier}/switchable_accounts"

    expect(response.status).to eq(200)
    # ログインしているスペースのアカウントは表示されないはず
    expect(response.body).not_to include("スペース1")
    expect(response.body).to include("スペース2")
  end
end
