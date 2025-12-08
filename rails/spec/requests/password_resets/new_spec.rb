# typed: false
# frozen_string_literal: true

RSpec.describe "GET /password_reset", type: :request do
  it "ログインしているとき、ホーム画面にリダイレクトすること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    get "/password_reset"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/home")
  end

  it "ログインしていないとき、パスワードリセットページが表示されること" do
    get "/password_reset"

    expect(response.status).to eq(200)
    expect(response.body).to include("パスワードを忘れた？")
  end
end
