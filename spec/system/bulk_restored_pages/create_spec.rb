# typed: false
# frozen_string_literal: true

RSpec.describe "ページの一括復元", type: :system do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space_record = create(:space_record, :small)

    visit trash_path(space_record.identifier)

    expect(page).to have_current_path("/sign_in")
  end

  it "別のスペースに参加しているとき、404エラーページが表示されること" do
    space_record = create(:space_record, :small)
    other_space_record = create(:space_record)
    user_record = create(:user_record, :with_password)
    create(:space_member_record, space_record: other_space_record, user_record:)

    sign_in(user_record:)

    visit trash_path(space_record.identifier)

    expect(page).to have_content("お探しのページは見つかりませんでした")
  end

  it "選択したページに問題があるとき、エラーメッセージを表示すること", :js do
    space_record = create(:space_record, :small)
    user_record = create(:user_record, :with_password)
    create(:space_member_record, space_record:, user_record:)
    topic_record = create(:topic_record, space_record:) # このトピックに参加していない
    page_record = create(:page_record, :trashed, space_record:, topic_record:)

    sign_in(user_record:)

    visit trash_path(space_record.identifier)

    # ページが表示されることを確認
    expect(page).to have_content(page_record.title)

    # チェックボックスを選択
    find("input[type='checkbox'][value='#{page_record.id}']").check

    # 復元ボタンをクリック
    find('button[type="submit"]', text: /Restore|復元/).click

    # エラーメッセージが表示されることを確認
    expect(page).to have_content("参加していないトピックのページが含まれているため復元できません")

    # ページがまだトラッシュページにいることを確認
    expect(page).to have_current_path(trash_path(space_record.identifier))
  end

  it "選択したページに問題がないとき、ページを復元できること", :js do
    space_record = create(:space_record, :small)
    user_record = create(:user_record, :with_password)
    space_member_record = create(:space_member_record, space_record:, user_record:)
    topic_record = create(:topic_record, space_record:)
    page_record = create(:page_record, :trashed, space_record:, topic_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)

    sign_in(user_record:)

    # ページがゴミ箱にあることを確認
    expect(page_record.trashed?).to be(true)

    visit trash_path(space_record.identifier)

    # ページが表示されることを確認
    expect(page).to have_content(page_record.title)
    expect(page).to have_content(topic_record.name)

    # チェックボックスを選択
    checkbox = find("input[type='checkbox'][value='#{page_record.id}']")
    checkbox.check
    expect(checkbox).to be_checked

    # 復元ボタンをクリック
    button = find('button[type="submit"]', text: /Restore|復元/)
    button.click

    # ページが復元されるまで待機（ページタイトルが消えるのを待つ）
    expect(page).not_to have_content(page_record.title)

    # 現在のパスを確認
    expect(page).to have_current_path(trash_path(space_record.identifier))

    # ページが復元されたことを確認
    page_record.reload
    expect(page_record.trashed?).to be(false)
  end

  it "複数のページを一括で復元できること", :js do
    space_record = create(:space_record, :small)
    user_record = create(:user_record, :with_password)
    space_member_record = create(:space_member_record, space_record:, user_record:)
    topic_record = create(:topic_record, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)

    # 複数のページを作成
    page_record1 = create(:page_record, :trashed, space_record:, topic_record:, title: "削除されたページ1")
    page_record2 = create(:page_record, :trashed, space_record:, topic_record:, title: "削除されたページ2")
    page_record3 = create(:page_record, :trashed, space_record:, topic_record:, title: "削除されたページ3")

    sign_in(user_record:)

    visit trash_path(space_record.identifier)

    # すべてのページが表示されることを確認
    expect(page).to have_content(page_record1.title)
    expect(page).to have_content(page_record2.title)
    expect(page).to have_content(page_record3.title)

    # 2つのページを選択
    check "pages_bulk_restoring_form[page_ids][]", option: page_record1.id
    check "pages_bulk_restoring_form[page_ids][]", option: page_record2.id

    # 復元ボタンをクリック
    find('button[type="submit"]', text: /Restore|復元/).click

    # 復元されたページがトラッシュから消えるのを待つ
    expect(page).not_to have_content(page_record1.title)
    expect(page).not_to have_content(page_record2.title)

    # トラッシュページにリダイレクトされることを確認
    expect(page).to have_current_path(trash_path(space_record.identifier))

    # 選択したページが復元されたことを確認
    page_record1.reload
    page_record2.reload
    page_record3.reload
    expect(page_record1.trashed?).to be(false)
    expect(page_record2.trashed?).to be(false)
    expect(page_record3.trashed?).to be(true) # 選択していないページは復元されない

    # 選択していないページは残っていることを確認
    expect(page).to have_content(page_record3.title)
  end

  it "ゴミ箱が空のとき、空の状態メッセージが表示されること" do
    space_record = create(:space_record, :small)
    user_record = create(:user_record, :with_password)
    create(:space_member_record, space_record:, user_record:)

    sign_in(user_record:)

    visit trash_path(space_record.identifier)

    # 空の状態メッセージが表示されることを確認
    expect(page).to have_content(I18n.t("messages.trash.empty_state_message"))

    # 復元ボタンが表示されないことを確認
    expect(page).not_to have_css('button[type="submit"]', text: /Restore|復元/)
  end
end
