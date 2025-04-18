# typed: false
# frozen_string_literal: true

RSpec.describe "POST /s/:space_identifier/settings/exports", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space_record, :small)

    post "/s/#{space.identifier}/settings/exports"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースに参加しているとき、404ページが表示されること" do
    user = create(:user_record, :with_password)

    space = create(:space_record, :small)

    other_space = create(:space_record)
    create(:space_member_record, space_record: other_space, user_record: user)

    sign_in(user_record: user)

    post "/s/#{space.identifier}/settings/exports"

    expect(response.status).to eq(404)
  end

  it "スペースに参加しているとき、エクスポートが作成できること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small, identifier: "space-identifier")
    create(:space_member_record, space_record: space, user_record: user)

    sign_in(user_record: user)

    expect(ExportRecord.count).to eq(0)

    post("/s/#{space.identifier}/settings/exports")

    expect(space.export_records.count).to eq(1)
    export = space.export_records.first

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/s/#{space.identifier}/settings/exports/#{export.id}")
  end
end
