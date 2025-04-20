# typed: false
# frozen_string_literal: true

RSpec.describe "POST /s/:space_identifier/topics", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)

    post "/s/#{space.identifier}/topics"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースに参加しているとき、404を返すこと" do
    user = create(:user, :with_password)
    space = create(:space, :small)
    other_space = create(:space)
    create(:space_member, user:, space: other_space)

    sign_in(user:)

    post "/s/#{space.identifier}/topics"

    expect(response.status).to eq(404)
  end

  it "スペースに参加している & 入力値が不正なとき、エラーメッセージを表示すること" do
    user = create(:user, :with_password)
    space = create(:space, :small)
    create(:space_member, user:, space:)

    sign_in(user:)

    expect(Topic.count).to eq(0)

    post("/s/#{space.identifier}/topics", params: {
      new_topic_form: {
        name: "", # トピック名が空
        description: "Topic Description",
        visibility: "public"
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("名前を入力してください")

    # バリデーションエラーになったのでトピックは作成されていないはず
    expect(Topic.count).to eq(0)
  end

  it "スペースに参加している & 入力値が正常なとき、トピックが作成できること" do
    user = create(:user, :with_password)
    space = create(:space, :small)
    create(:space_member, user:, space:)

    sign_in(user:)

    expect(Topic.count).to eq(0)

    post("/s/#{space.identifier}/topics", params: {
      new_topic_form: {
        name: "テストトピック",
        description: "テストトピックです",
        visibility: "public"
      }
    })

    expect(response.status).to eq(302)

    expect(Topic.count).to eq(1)
    topic = Topic.first
    expect(response).to redirect_to("/s/#{space.identifier}/topics/#{topic.number}")
  end
end
