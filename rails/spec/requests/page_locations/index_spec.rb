# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/page_locations", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space_record = create(:space_record)

    get "/s/#{space_record.identifier}/page_locations"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースに参加しているとき、404ページが表示されること" do
    space_record = create(:space_record)

    other_space_record = create(:space_record)
    user_record = create(:user_record, :with_password)
    create(:space_member_record, space_record: other_space_record, user_record:)

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/page_locations"

    expect(response.status).to eq(404)
  end

  it "スペースに参加している & キーワードが渡されなかったとき、空のリストを返すこと" do
    space_record = create(:space_record)
    user_record = create(:user_record, :with_password)
    create(:space_member_record, space_record:, user_record:)

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/page_locations"

    expect(response.status).to eq(200)

    actual = JSON.parse(response.body)
    expect(actual["page_locations"]).to eq([])
  end

  it "スペースに参加している & キーワードが1つ渡されたとき、キーワードにマッチするタイトルを返すこと" do
    space_record = create(:space_record)
    user_record = create(:user_record, :with_password)
    create(:space_member_record, space_record:, user_record:)

    page_record_1 = create(:page_record, :published, space_record:, title: "aaabbbccc", modified_at: 1.day.ago)
    page_record_2 = create(:page_record, :published, space_record:, title: "aaabbbddd", modified_at: 2.days.ago)
    page_record_3 = create(:page_record, :published, space_record:, title: "aaabbbeee", modified_at: 3.days.ago)

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/page_locations?q=aaa"

    expect(response.status).to eq(200)

    actual = JSON.parse(response.body)
    expect(actual["page_locations"]).to eq([
      {
        "key" => "#{page_record_1.topic_record.name}/#{page_record_1.title}"
      },
      {
        "key" => "#{page_record_2.topic_record.name}/#{page_record_2.title}"
      },
      {
        "key" => "#{page_record_3.topic_record.name}/#{page_record_3.title}"
      }
    ])
  end

  it "スペースに参加している & キーワードが2つ渡されたとき、2つのキーワードに全てマッチするタイトルを返すこと" do
    space_record = create(:space_record)
    user_record = create(:user_record, :with_password)
    create(:space_member_record, space_record:, user_record:)

    _page_record_1 = create(:page_record, :published, space_record:, title: "aaabbbccc", modified_at: 1.day.ago)
    page_record_2 = create(:page_record, :published, space_record:, title: "aaabbbddd", modified_at: 2.days.ago)
    _page_record_3 = create(:page_record, :published, space_record:, title: "aaabbbeee", modified_at: 3.days.ago)

    sign_in(user_record:)

    get "/s/#{space_record.identifier}/page_locations?q=aaa%20ddd"

    expect(response.status).to eq(200)

    actual = JSON.parse(response.body)
    expect(actual["page_locations"]).to eq([
      {
        "key" => "#{page_record_2.topic_record.name}/#{page_record_2.title}"
      }
    ])
  end
end
