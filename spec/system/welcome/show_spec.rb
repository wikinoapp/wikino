# typed: false
# frozen_string_literal: true

RSpec.describe "トップページ", type: :system do
  it do
    visit "/"

    expect(page).to have_content "Markdownで書き、リンクで見つかるWikiアプリ"
  end
end
