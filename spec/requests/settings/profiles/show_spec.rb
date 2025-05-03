# typed: false
# frozen_string_literal: true

RSpec.describe "GET /settings/profile", type: :request do
  it "ログインしていないとき、ログインページが表示されること" do
    get "/settings/profile"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしているとき、プロフィール設定ページが表示されること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    get "/settings/profile"

    expect(response.status).to eq(200)
    expect(response.body).to include("プロフィール編集")
  end
end
