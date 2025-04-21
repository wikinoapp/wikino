# typed: false
# frozen_string_literal: true

RSpec.describe "GET /spaces/new", type: :request do
  it "ログインしているとき、スペース作成画面が表示されること" do
    user = create(:user_record, :with_password)

    sign_in(user_record: user)

    get "/spaces/new"
    page = Capybara.string(response.body)

    expect(response.status).to eq(200)
    expect(page).to have_title("新規スペース")
  end

  it "ログインしていないとき、ログインページが表示されること" do
    get "/spaces/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end
end
