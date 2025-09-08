# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "Global Hotkey", type: :system do
  it "検索ページ以外でsキーまたは/キーを押すと検索ページに遷移すること", :js do
    user_record = create(:user_record, :with_password)
    sign_in(user_record:)

    visit root_path

    # sキーを押すと検索ページに遷移
    page.driver.browser.action.send_keys("s").perform
    expect(page).to have_current_path(search_path)

    # ホームページに戻る
    visit root_path

    # /キーを押すと検索ページに遷移
    page.driver.browser.action.send_keys("/").perform
    expect(page).to have_current_path(search_path)
  end

  it "スペース内でsキーまたは/キーを押すとspace:フィルターが付与された検索ページに遷移すること", :js do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    create(:space_member_record, user_record:, space_record:)
    sign_in(user_record:)

    visit space_path(space_record.identifier)

    # sキーを押すとspace:フィルターが付与された検索ページに遷移
    page.driver.browser.action.send_keys("s").perform
    expect(page).to have_current_path(search_path(q: "space:test-space"))

    # スペースページに戻る
    visit space_path(space_record.identifier)

    # /キーを押すとspace:フィルターが付与された検索ページに遷移
    page.driver.browser.action.send_keys("/").perform
    expect(page).to have_current_path(search_path(q: "space:test-space"))
  end

  it "トピック内でsキーまたは/キーを押すとspace:フィルターが付与された検索ページに遷移すること", :js do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    create(:space_member_record, user_record:, space_record:)
    sign_in(user_record:)

    visit topic_path(space_record.identifier, topic_record.number)

    # sキーを押すとspace:フィルターが付与された検索ページに遷移
    page.driver.browser.action.send_keys("s").perform
    expect(page).to have_current_path(search_path(q: "space:test-space"))

    # トピックページに戻る
    visit topic_path(space_record.identifier, topic_record.number)

    # /キーを押すとspace:フィルターが付与された検索ページに遷移
    page.driver.browser.action.send_keys("/").perform
    expect(page).to have_current_path(search_path(q: "space:test-space"))
  end

  it "ページ内でsキーまたは/キーを押すとspace:フィルターが付与された検索ページに遷移すること", :js do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    create(:space_member_record, user_record:, space_record:)
    page_record = create(:page_record, :published, space_record:, topic_record:)
    sign_in(user_record:)

    visit page_path(space_record.identifier, page_record.number)

    # sキーを押すとspace:フィルターが付与された検索ページに遷移
    page.driver.browser.action.send_keys("s").perform
    expect(page).to have_current_path(search_path(q: "space:test-space"))

    # ページに戻る
    visit page_path(space_record.identifier, page_record.number)

    # /キーを押すとspace:フィルターが付与された検索ページに遷移
    page.driver.browser.action.send_keys("/").perform
    expect(page).to have_current_path(search_path(q: "space:test-space"))
  end

  it "ページ編集画面でsキーまたは/キーを押すとspace:フィルターが付与された検索ページに遷移すること", :js do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)
    page_record = create(:page_record, :published, space_record:, topic_record:)
    sign_in(user_record:)

    visit edit_page_path(space_record.identifier, page_record.number)

    # エディタ外の要素にフォーカスを移す
    find("h1").click

    # sキーを押すとspace:フィルターが付与された検索ページに遷移
    page.driver.browser.action.send_keys("s").perform
    expect(page).to have_current_path(search_path(q: "space:test-space"))

    # ページ編集画面に戻る
    visit edit_page_path(space_record.identifier, page_record.number)

    # エディタ外の要素にフォーカスを移す
    find("h1").click

    # /キーを押すとspace:フィルターが付与された検索ページに遷移
    page.driver.browser.action.send_keys("/").perform
    expect(page).to have_current_path(search_path(q: "space:test-space"))
  end
end
