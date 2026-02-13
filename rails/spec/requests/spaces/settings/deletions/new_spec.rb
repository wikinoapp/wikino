# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/settings/deletion/new", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space_record = create(:space_record, :small)

    get "/s/#{space_record.identifier}/settings/deletion/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & スペースに参加していないとき、404を返すこと" do
    space_record = create(:space_record, :small)
    other_space_record = create(:space_record)
    user_record = create(:user_record, :with_password)
    create(:space_member_record, :owner, space_record: other_space_record, user_record:)

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/settings/deletion/new"

    expect(response.status).to eq(404)
  end

  it "ログインしている & 別のスペースに参加しているとき、404を返すこと" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, :small)
    other_space_record = create(:space_record)
    create(:space_member_record, user_record:, space_record: other_space_record)

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/settings/deletion/new"

    expect(response.status).to eq(404)
  end

  it "ログインしている & スペースに参加している & スペースのオーナーのとき、削除確認画面が表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, :small, name: "テストスペース")
    create(:space_member_record, :owner, space_record:, user_record:)

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/settings/deletion/new"

    expect(response.status).to eq(200)
    expect(response.body).to include("テストスペース")
    expect(response.body).to include("スペースを削除")
  end
end
