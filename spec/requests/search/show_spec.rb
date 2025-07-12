# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "GET /search", type: :request do
  let(:user_record) { create(:user_record, :with_password) }
  let(:space_record) { create(:space_record) }
  let(:topic_record) { create(:topic_record, space_record:) }
  let!(:space_membership) { create(:space_member_record, user_record:, space_record:) }

  before do
    sign_in(user_record:)
  end

  it "検索ページが正常に表示されること" do
    get search_path
    expect(response).to have_http_status(:ok)
  end

  it "ログインしていない場合、ログインページにリダイレクトされること" do
    delete user_session_path
    get search_path
    expect(response).to redirect_to(sign_in_path)
  end

  it "検索キーワードがない場合、検索フォームのみが表示されること" do
    get search_path
    expect(response).to have_http_status(:ok)
    expect(response.body).to include("検索")
  end

  it "有効な検索キーワードがある場合、検索が実行されること" do
    create(:page_record, 
      space_record:, 
      topic_record:, 
      title: "テストページ"
    )

    get search_path, params: { q: "テスト" }
    
    expect(response).to have_http_status(:ok)
    expect(response.body).to include("テストページ")
  end

  it "検索結果がない場合、適切なメッセージが表示されること" do
    get search_path, params: { q: "存在しないキーワード" }
    
    expect(response).to have_http_status(:ok)
    expect(response.body).to include("検索結果が見つかりませんでした")
  end

  it "無効な検索キーワードの場合、エラーメッセージが表示されること" do
    get search_path, params: { q: "a" }
    
    expect(response).to have_http_status(:ok)
    expect(response.body).to include("は2文字以上で入力してください")
  end

  it "他のユーザーのスペースのページは検索結果に含まれないこと" do
    other_user_record = create(:user_record, :with_password)
    other_space_record = create(:space_record)
    other_topic_record = create(:topic_record, space_record: other_space_record)
    create(:space_member_record, user_record: other_user_record, space_record: other_space_record)
    create(:page_record, 
      space_record: other_space_record, 
      topic_record: other_topic_record, 
      title: "他のユーザーのページ"
    )

    get search_path, params: { q: "他のユーザー" }
    
    expect(response).to have_http_status(:ok)
    expect(response.body).to include("検索結果が見つかりませんでした")
  end
end