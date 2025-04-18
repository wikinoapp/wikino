# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /s/:space_identifier/topics/:topic_number/settings/general", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space_record, :small)
    topic = create(:topic_record, space_record: space)

    patch "/s/#{space.identifier}/topics/#{topic.number}/settings/general"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & スペースに参加していないとき、404を返すこと" do
    user = create(:user_record, :with_password)
    space = create(:space_record)
    topic = create(:topic_record, space_record: space)

    sign_in(user_record: user)

    patch "/s/#{space.identifier}/topics/#{topic.number}/settings/general"

    expect(response.status).to eq(404)
  end

  it "ログインしている & 別のスペースに参加しているとき、404ページが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    topic = create(:topic_record, space_record: space)
    other_space = create(:space_record)
    create(:space_member_record, space_record: other_space, user_record: user)

    sign_in(user_record: user)

    patch "/s/#{space.identifier}/topics/#{topic.number}/settings/general"

    expect(response.status).to eq(404)
  end

  it "ログインしている & スペースに参加している & 入力値が不正なとき、エラーメッセージを表示すること" do
    user = create(:user_record, :with_password)
    space = create(:space_record)
    topic = create(:topic_record, space_record: space, name: "Before Name")
    create(:space_member_record, space_record: space, user_record: user)

    sign_in(user_record: user)

    expect(topic.name).to eq("Before Name")

    patch("/s/#{space.identifier}/topics/#{topic.number}/settings/general", params: {
      edit_topic_form: {
        name: "", # トピック名が空
        description: "Updated Description",
        visibility: "public"
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("名前を入力してください")

    # バリデーションエラーになったのでトピックは更新されていないはず
    expect(topic.reload.name).to eq("Before Name")
  end

  it "ログインしている & スペースに参加している & 入力値が正しいとき、トピックが更新できること" do
    user = create(:user_record, :with_password)
    space = create(:space_record)
    topic = create(:topic_record, space_record: space, name: "Before Name", description: "Before Description")
    create(:space_member_record, space_record: space, user_record: user)

    sign_in(user_record: user)

    expect(topic.name).to eq("Before Name")
    expect(topic.description).to eq("Before Description")

    patch("/s/#{space.identifier}/topics/#{topic.number}/settings/general", params: {
      edit_topic_form: {
        name: "Updated Name",
        description: "Updated Description",
        visibility: "public"
      }
    })

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/s/#{space.identifier}/topics/#{topic.number}/settings/general")

    expect(topic.reload.name).to eq("Updated Name")
    expect(topic.description).to eq("Updated Description")
    expect(topic.visibility).to eq("public")
  end
end
