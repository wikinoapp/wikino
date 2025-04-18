# typed: false
# frozen_string_literal: true

RSpec.describe "GET /home", type: :request do
  it "ログインしているとき、ホーム画面が表示されること" do
    user = create(:user_record, :with_password)

    sign_in(user:)

    get "/home"

    expect(response.status).to eq(200)
    expect(response.body).to include("ホーム")
  end

  it "ログインしていないとき、ログインページが表示されること" do
    get "/home"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end
end
