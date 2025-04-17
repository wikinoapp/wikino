# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/settings/exports/:export_id", type: :request do
  it "ログインしていないとき、ログインページが表示されること" do
    space = create(:space_record)
    export = create(:export_record, space_record: space)

    get "/s/#{space.identifier}/settings/exports/#{export.id}"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & スペースに参加していないとき、404を返すこと" do
    user = create(:user_record, :with_password)
    space = create(:space_record)
    export = create(:export_record, space_record: space)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/settings/exports/#{export.id}"

    expect(response.status).to eq(404)
  end

  it "ログインしている & 別のスペースに参加しているとき、404を返すこと" do
    user = create(:user_record, :with_password)
    space = create(:space_record)
    other_space = create(:space_record)
    create(:space_member_record, user_record: user, space_record: other_space)
    export = create(:export_record, space_record: space)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/settings/exports/#{export.id}"

    expect(response.status).to eq(404)
  end

  it "ログインしている & スペースに参加しているとき、エクスポート画面が表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record)
    create(:space_member_record, user_record: user, space_record: space)
    export = create(:export_record, space_record: space)
    create(:export_status_record, space_record: space, export_record: export)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/settings/exports/#{export.id}"

    expect(response.status).to eq(200)
    expect(response.body).to include("エクスポート")
  end
end
