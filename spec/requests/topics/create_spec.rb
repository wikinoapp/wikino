# typed: false
# frozen_string_literal: true

RSpec.describe "POST /s/:space_identifier/topics", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space_record, :small)

    post "/s/#{space.identifier}/topics"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースに参加しているとき、404を返すこと" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    other_space = create(:space_record)
    create(:space_member_record, user_record: user, space_record: other_space)

    sign_in(user_record: user)

    post "/s/#{space.identifier}/topics"

    expect(response.status).to eq(404)
  end

  it "スペースに参加している & 入力値が不正なとき、エラーメッセージを表示すること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    create(:space_member_record, user_record: user, space_record: space)

    sign_in(user_record: user)

    expect(TopicRecord.count).to eq(0)

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
    expect(TopicRecord.count).to eq(0)
  end

  it "スペースに参加している & 非公開トピックを作成しようとしたとき、エラーメッセージを表示すること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    create(:space_member_record, user_record: user, space_record: space)

    sign_in(user_record: user)

    expect(TopicRecord.count).to eq(0)

    post("/s/#{space.identifier}/topics", params: {
      new_topic_form: {
        name: "テストトピック",
        description: "テストトピックです",
        visibility: "private"
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("公開設定は一覧にありません")

    # バリデーションエラーになったのでトピックは作成されていないはず
    expect(TopicRecord.count).to eq(0)
  end

  it "スペースに参加している & 入力値が正常なとき、トピックが作成できること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    create(:space_member_record, user_record: user, space_record: space)

    sign_in(user_record: user)

    expect(TopicRecord.count).to eq(0)

    post("/s/#{space.identifier}/topics", params: {
      new_topic_form: {
        name: "テストトピック",
        description: "テストトピックです",
        visibility: "public"
      }
    })

    expect(response.status).to eq(302)

    expect(TopicRecord.count).to eq(1)
    topic = TopicRecord.first
    expect(response).to redirect_to("/s/#{space.identifier}/topics/#{topic.number}")
  end
end
