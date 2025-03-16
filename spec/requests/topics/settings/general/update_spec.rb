# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /s/:space_identifier/topics/:topic_number/settings/general", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    topic = create(:topic, space:)

    patch "/s/#{space.identifier}/topics/#{topic.number}/settings/general"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & スペースに参加していないとき、404を返すこと" do
    user = create(:user, :with_password)
    space = create(:space)
    topic = create(:topic, space:)

    sign_in(user:)

    patch "/s/#{space.identifier}/topics/#{topic.number}/settings/general"

    expect(response.status).to eq(404)
  end

  it "ログインしている & 別のスペースに参加しているとき、404ページが表示されること" do
    user = create(:user, :with_password)
    space = create(:space, :small)
    topic = create(:topic, space:)
    other_space = create(:space)
    create(:space_member, space: other_space, user:)

    sign_in(user:)

    patch "/s/#{space.identifier}/topics/#{topic.number}/settings/general"

    expect(response.status).to eq(404)
  end

  it "ログインしている & スペースに参加している & 入力値が不正なとき、エラーメッセージを表示すること" do
    user = create(:user, :with_password)
    space = create(:space)
    topic = create(:topic, space:, name: "Before Name")
    create(:space_member, space:, user:)

    sign_in(user:)

    expect(topic.name).to eq("Before Name")

    patch("/s/#{space.identifier}/topics/#{topic.number}/settings/general", params: {
      edit_topic_form: {
        name: "",
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
    user = create(:user, :with_password)
    space = create(:space)
    topic = create(:topic, space:, name: "Before Name", description: "Before Description")
    create(:space_member, space:, user:)

    sign_in(user:)

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
