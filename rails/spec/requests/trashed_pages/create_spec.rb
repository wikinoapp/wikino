# typed: false
# frozen_string_literal: true

RSpec.describe "POST /s/:space_identifier/pages/:page_number/trash", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space_record, :small)
    topic = create(:topic_record, space_record: space)
    page = create(:page_record, topic_record: topic)

    post "/s/#{space.identifier}/pages/#{page.number}/trash"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "スペースに参加していないとき、404を返すこと" do
    space = create(:space_record, :small)
    topic = create(:topic_record, space_record: space)
    page = create(:page_record, topic_record: topic)

    other_space = create(:space_record)
    user = create(:user_record, :with_password)
    create(:space_member_record, :owner, space_record: other_space, user_record: user)

    sign_in(user_record: user)

    post "/s/#{space.identifier}/pages/#{page.number}/trash"

    expect(response.status).to eq(404)
  end

  it "指定したページが存在しないとき、エラーメッセージを表示すること" do
    space = create(:space_record, :small)
    user = create(:user_record, :with_password)
    create(:space_member_record, :owner, space_record: space, user_record: user)

    sign_in(user_record: user)

    post "/s/#{space.identifier}/pages/0/trash"

    expect(response.status).to eq(404)
  end

  it "オーナーとしてログインしているとき、ゴミ箱に移動できること" do
    space = create(:space_record, :small)
    user = create(:user_record, :with_password)
    space_member = create(:space_member_record, :owner, space_record: space, user_record: user)
    topic = create(:topic_record, space_record: space)
    page = create(:page_record, space_record: space, topic_record: topic)
    create(
      :topic_member_record,
      space_record: space,
      topic_record: topic,
      space_member_record: space_member
    )

    sign_in(user_record: user)

    expect(page.trashed?).to be(false)

    post "/s/#{space.identifier}/pages/#{page.number}/trash"

    expect(response.status).to eq(302)
    expect(page.reload.trashed?).to be(true)
  end
end
