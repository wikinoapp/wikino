# typed: false
# frozen_string_literal: true

RSpec.describe "GET /sign_up", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user = create(:user, :with_password)

    sign_in(user:)

    get "/sign_up"

    expect(response.status).to eq(302)
    space = user.space
    expect(response).to redirect_to("/s/#{space.identifier}")
  end

  it "ログインしている & `skip_no_authentication` が付与されているとき、アカウント作成ページが表示されること" do
    user = create(:user, :with_password)

    sign_in(user:)

    get "/sign_in?skip_no_authentication=true"

    expect(response.status).to eq(200)
    expect(response.body).to include("アカウント作成")
  end

  it "ログインしていないとき、アカウント作成ページが表示されること" do
    get "/sign_up"

    expect(response.status).to eq(200)
    expect(response.body).to include("アカウント作成")
  end
end
