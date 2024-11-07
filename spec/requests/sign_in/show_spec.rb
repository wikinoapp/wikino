# typed: false
# frozen_string_literal: true

RSpec.describe "GET /sign_in", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user = create(:user, :with_password)

    sign_in(user:)

    get "/sign_in"

    expect(response).to have_http_status(:found)
    space = user.space
    expect(response).to redirect_to(space_path(space.identifier))
  end

  it "ログインしている & `skip_no_authentication` が付与されているとき、ログインページが表示されること" do
    user = create(:user, :with_password)

    sign_in(user:)

    get "/sign_in?skip_no_authentication=true"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("ログイン")
  end

  it "ログインしていないとき、ログインページが表示されること" do
    get "/sign_in"

    expect(response).to have_http_status(:ok)
    expect(response.body).to include("ログイン")
  end
end
