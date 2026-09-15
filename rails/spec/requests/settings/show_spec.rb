# typed: false
# frozen_string_literal: true

RSpec.describe "GET /settings", type: :request do
  it "ログインしているとき、設定ページが表示されること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    get "/settings"

    expect(response.status).to eq(200)
    expect(response.body).to include("設定")
  end

  it "ヘッダーと本文が同じコンテナ幅で描画されること" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    get "/settings"

    doc = Nokogiri::HTML(response.body)
    width_class = "max-w-(--content-screen-max-width-small)"

    expect(doc.at_css("header > div")["class"]).to include(width_class)
    expect(doc.at_css("main#main > div")["class"]).to include(width_class)
  end

  it "main ランドマークが入れ子にならないこと" do
    user_record = create(:user_record, :with_password)

    sign_in(user_record:)

    get "/settings"

    doc = Nokogiri::HTML(response.body)

    expect(doc.css("main").size).to eq(1)
  end

  it "ログインしていないとき、ログインページが表示されること" do
    get "/settings"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end
end
