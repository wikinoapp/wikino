# typed: false
# frozen_string_literal: true

RSpec.describe "GET /", type: :request do
  it "ログインしているとき、ランディングページが表示されること" do
    user = create(:user, :with_password)
    sign_in(user:)

    get "/"

    expect(response.status).to eq(200)
    expect(response.body).to include("Wikinoにようこそ！")
  end

  it "ログインしていないとき、ランディングページが表示されること" do
    get "/"

    expect(response.status).to eq(200)
    expect(response.body).to include("Wikinoにようこそ！")
  end
end
