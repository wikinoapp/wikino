# typed: false
# frozen_string_literal: true

RSpec.describe "GET /settings/account/deletion/new", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    get "/settings/account/deletion/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & アクティブなスペースがあるとき、削除できない旨が表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    create(:space_member_record, space_record:, user_record:)

    sign_in(user_record:)

    get "/settings/account/deletion/new"

    expect(response.status).to eq(200)
    expect(response.body).to include("このアカウントは削除要件を満たしていません")
  end

  it "ログインしている & アクティブなスペースがないとき、削除確認画面が表示されること" do
    user_record = create(:user_record, :with_password, atname: "test_user")

    sign_in(user_record:)

    get "/settings/account/deletion/new"

    expect(response.status).to eq(200)
    expect(response.body).to include("test_user")
    expect(response.body).to include("アカウントを削除")
  end
end
