# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/topics/:topic_number/edit_suggestions", type: :request do
  it "ログインしていない & 非公開トピックのとき、404を返すこと" do
    space_record = FactoryBot.create(:space_record, :small)
    topic_record = FactoryBot.create(:topic_record, :private, space_record:)

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions"

    expect(response.status).to eq(404)
  end

  it "別のスペースに参加している & 非公開トピックのとき、404を返すこと" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record, :small)
    topic_record = FactoryBot.create(:topic_record, :private, space_record:)

    other_space_record = FactoryBot.create(:space_record)
    FactoryBot.create(:space_member_record, user_record:, space_record: other_space_record)

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions"

    expect(response.status).to eq(404)
  end

  it "ログインしていない & 公開トピックのとき、編集提案一覧が表示されること" do
    space_record = FactoryBot.create(:space_record, :small)
    topic_record = FactoryBot.create(:topic_record, :public, space_record:, name: "公開されているトピック")

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions"

    expect(response.status).to eq(200)
    expect(response.body).to include("編集提案")
    expect(response.body).to include("公開されているトピック")
  end

  it "スペースに参加している & 参加している公開トピックのとき、編集提案一覧が表示されること" do
    space_record = FactoryBot.create(:space_record, :small)
    user_record = FactoryBot.create(:user_record, :with_password)
    space_member_record = FactoryBot.create(:space_member_record, :owner, space_record:, user_record:)
    topic_record = FactoryBot.create(:topic_record, :public, space_record:, name: "公開されているトピック")
    FactoryBot.create(:topic_member_record, topic_record:, space_member_record:)

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions"

    expect(response.status).to eq(200)
    expect(response.body).to include("編集提案")
    expect(response.body).to include("公開されているトピック")
  end

  it "スペースに参加している & 参加している非公開トピックのとき、編集提案一覧が表示されること" do
    space_record = FactoryBot.create(:space_record, :small)
    user_record = FactoryBot.create(:user_record, :with_password)
    space_member_record = FactoryBot.create(:space_member_record, :owner, space_record:, user_record:)
    topic_record = FactoryBot.create(:topic_record, :private, space_record:, name: "非公開トピック")
    FactoryBot.create(:topic_member_record, topic_record:, space_member_record:)

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions"

    expect(response.status).to eq(200)
    expect(response.body).to include("編集提案")
    expect(response.body).to include("非公開トピック")
  end

  it "オープンフィルターで編集提案が表示されること" do
    space_record = FactoryBot.create(:space_record, :small)
    user_record = FactoryBot.create(:user_record, :with_password)
    space_member_record = FactoryBot.create(:space_member_record, :owner, space_record:, user_record:)
    topic_record = FactoryBot.create(:topic_record, :public, space_record:)

    # オープンステータスの編集提案を作成
    FactoryBot.create(
      :edit_suggestion_record,
      :open,
      topic_record:,
      space_record:,
      created_space_member_record: space_member_record,
      title: "オープンな編集提案"
    )

    # クローズステータスの編集提案を作成
    FactoryBot.create(
      :edit_suggestion_record,
      :closed,
      topic_record:,
      space_record:,
      created_space_member_record: space_member_record,
      title: "クローズした編集提案"
    )

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions?state=open"

    expect(response.status).to eq(200)
    expect(response.body).to include("オープンな編集提案")
    expect(response.body).not_to include("クローズした編集提案")
  end

  it "クローズフィルターで編集提案が表示されること" do
    space_record = FactoryBot.create(:space_record, :small)
    user_record = FactoryBot.create(:user_record, :with_password)
    space_member_record = FactoryBot.create(:space_member_record, :owner, space_record:, user_record:)
    topic_record = FactoryBot.create(:topic_record, :public, space_record:)

    # オープンステータスの編集提案を作成
    FactoryBot.create(
      :edit_suggestion_record,
      :open,
      topic_record:,
      space_record:,
      created_space_member_record: space_member_record,
      title: "オープンな編集提案"
    )

    # クローズステータスの編集提案を作成
    FactoryBot.create(
      :edit_suggestion_record,
      :closed,
      topic_record:,
      space_record:,
      created_space_member_record: space_member_record,
      title: "クローズした編集提案"
    )

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions?state=closed"

    expect(response.status).to eq(200)
    expect(response.body).to include("クローズした編集提案")
    expect(response.body).not_to include("オープンな編集提案")
  end

  it "デフォルトでオープンフィルターが適用されること" do
    space_record = FactoryBot.create(:space_record, :small)
    user_record = FactoryBot.create(:user_record, :with_password)
    space_member_record = FactoryBot.create(:space_member_record, :owner, space_record:, user_record:)
    topic_record = FactoryBot.create(:topic_record, :public, space_record:)

    # 下書きステータスの編集提案を作成
    FactoryBot.create(
      :edit_suggestion_record,
      :draft,
      topic_record:,
      space_record:,
      created_space_member_record: space_member_record,
      title: "下書きの編集提案"
    )

    # 反映済みステータスの編集提案を作成
    FactoryBot.create(
      :edit_suggestion_record,
      :applied,
      topic_record:,
      space_record:,
      created_space_member_record: space_member_record,
      title: "反映済みの編集提案"
    )

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/topics/#{topic_record.number}/edit_suggestions"

    expect(response.status).to eq(200)
    expect(response.body).to include("下書きの編集提案")
    expect(response.body).not_to include("反映済みの編集提案")
  end
end
