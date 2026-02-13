# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/topics/new", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space_record, :small)

    get "/s/#{space.identifier}/topics/new"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースに参加しているとき、404を返すこと" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)

    other_space = create(:space_record)
    create(:space_member_record, user_record: user, space_record: other_space)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/topics/new"

    expect(response.status).to eq(404)
  end

  it "スペースに参加しているとき、トピック作成ページが表示されること" do
    space = create(:space_record, :small)
    user = create(:user_record, :with_password)
    create(:space_member_record, user_record: user, space_record: space)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/topics/new"

    expect(response.status).to eq(200)
    expect(response.body).to include("新規トピック")
  end
end
