# typed: false
# frozen_string_literal: true

RSpec.describe "GET /settings/email", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトされること" do
    get "/settings/email"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしているとき、メールアドレス変更ページが表示されること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    get "/settings/email"

    expect(response.status).to eq(200)
    expect(response.body).to include("メールアドレスの変更")
  end
end
