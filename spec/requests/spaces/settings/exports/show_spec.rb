# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/settings/exports/:export_id", type: :request do
  it "ログインしていないとき、ログインページが表示されること" do
    space = create(:space)
    export = create(:export, space:)

    get "/s/#{space.identifier}/settings/exports/#{export.id}"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & スペースに参加していないとき、404を返すこと" do
    user = create(:user, :with_password)
    space = create(:space)
    export = create(:export, space:)

    sign_in(user:)

    get "/s/#{space.identifier}/settings/exports/#{export.id}"

    expect(response.status).to eq(404)
  end

  it "ログインしている & 別のスペースに参加しているとき、404を返すこと" do
    user = create(:user, :with_password)
    space = create(:space)
    other_space = create(:space)
    create(:space_member, user:, space: other_space)
    export = create(:export, space:)

    sign_in(user:)

    get "/s/#{space.identifier}/settings/exports/#{export.id}"

    expect(response.status).to eq(404)
  end

  it "ログインしている & スペースに参加しているとき、エクスポート画面が表示されること" do
    user = create(:user, :with_password)
    space = create(:space)
    create(:space_member, user:, space:)
    export = create(:export, space:)

    sign_in(user:)

    get "/s/#{space.identifier}/settings/exports/#{export.id}"

    expect(response.status).to eq(200)
    expect(response.body).to include("エクスポート")
  end
end
