# typed: false
# frozen_string_literal: true

RSpec.describe "POST /s/:space_identifier/topics", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)

    post "/s/#{space.identifier}/topics"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースにログインしているとき、ログインページにリダイレクトすること" do
    space = create(:space, :small)
    other_space = create(:space)
    user = create(:user, :with_password, space: other_space)

    sign_in(user:)

    post "/s/#{space.identifier}/topics"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "入力値が不正なとき、エラーメッセージを表示すること" do
    space = create(:space, :small)
    user = create(:user, :with_password, space:)

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

  it "オーナーとしてログインしているとき、トピックが作成できること" do
    space = create(:space, :small)
    user = create(:user, :owner, :with_password, space:)

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
