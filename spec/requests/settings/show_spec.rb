# typed: false
# frozen_string_literal: true

RSpec.describe "GET /settings", type: :request do
  it "ログインしているとき、設定ページが表示されること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    get "/settings"

    expect(response.status).to eq(200)
    expect(response.body).to include("設定")
  end

  it "ログインしていないとき、ログインページが表示されること" do
    get "/settings"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end
end
