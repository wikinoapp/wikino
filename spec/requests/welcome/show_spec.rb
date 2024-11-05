# typed: false
# frozen_string_literal: true

RSpec.describe "GET /", type: :request do
  it "ログインしているときはホーム画面にリダイレクトすること" do
    user = create(:user, :with_password)

    sign_in(user:)

    get "/"

    space = user.space
    expect(response).to redirect_to(space_path(space.identifier))
  end

  it "ログインしていないときはランディングページが表示されること" do
    get "/"

    expect(response.body).to include("Wikinoにようこそ！")
  end
end
