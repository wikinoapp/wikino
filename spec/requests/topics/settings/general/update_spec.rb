# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /s/:space_identifier/topics/:topic_number/settings/general", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space_record = create(:space_record, :small)
    topic_record = create(:topic_record, space_record:)

    patch "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings/general"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & スペースに参加していないとき、404を返すこと" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    topic_record = create(:topic_record, space_record:)

    sign_in(user_record:)

    patch "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings/general"

    expect(response.status).to eq(404)
  end

  it "ログインしている & 別のスペースに参加しているとき、404ページが表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, :small)
    topic_record = create(:topic_record, space_record:)
    other_space_record = create(:space_record)
    create(:space_member_record, space_record: other_space_record, user_record:)

    sign_in(user_record:)

    patch "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings/general"

    expect(response.status).to eq(404)
  end

  it "ログインしている & スペースに参加している & トピックに参加していないとき、404を返すこと" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    topic_record = create(:topic_record, space_record:)
    create(:space_member_record, space_record:, user_record:)

    sign_in(user_record:)

    patch "/s/#{space_record.identifier}/topics/#{topic_record.number}/settings/general"

    expect(response.status).to eq(404)
  end

  it "ログインしている & スペースに参加している & トピックに参加している & 入力値が不正なとき、エラーメッセージを表示すること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    topic_record = create(:topic_record, space_record:, name: "Before Name")
    space_member_record = create(:space_member_record, space_record:, user_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)

    sign_in(user_record:)

    expect(topic_record.name).to eq("Before Name")

    patch("/s/#{space_record.identifier}/topics/#{topic_record.number}/settings/general", params: {
      topic_form_edit: {
        name: "", # トピック名が空
        description: "Updated Description",
        visibility: "public"
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("名前を入力してください")

    # バリデーションエラーになったのでトピックは更新されていないはず
    expect(topic_record.reload.name).to eq("Before Name")
  end

  it "ログインしている & スペースに参加している & トピックに参加している & 入力値が正しいとき、トピックが更新できること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    topic_record = create(:topic_record, space_record:, name: "Before Name", description: "Before Description")
    space_member_record = create(:space_member_record, space_record:, user_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)

    sign_in(user_record:)

    expect(topic_record.name).to eq("Before Name")
    expect(topic_record.description).to eq("Before Description")

    patch("/s/#{space_record.identifier}/topics/#{topic_record.number}/settings/general", params: {
      topic_form_edit: {
        name: "Updated Name",
        description: "Updated Description",
        visibility: "public"
      }
    })

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/s/#{space_record.identifier}/topics/#{topic_record.number}/settings/general")

    expect(topic_record.reload.name).to eq("Updated Name")
    expect(topic_record.description).to eq("Updated Description")
    expect(topic_record.visibility).to eq("public")
  end
end
