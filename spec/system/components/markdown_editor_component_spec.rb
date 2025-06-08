# typed: false
# frozen_string_literal: true

RSpec.describe "Markdownエディター", type: :system do
  describe "Wikiリンクの補完候補" do
    it "Wikiリンクの補完候補が表示されること" do
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record)
      page_record = create(:page_record, space_record:)
      topic_record = page_record.topic_record
      space_member_record = create(:space_member_record, space_record:, user_record:)
      create(:topic_member_record, space_record:, topic_record:, space_member_record:)
      create(:page_record, :published, space_record:, topic_record:, title: "Other Page 1")
      create(:page_record, :published, space_record:, topic_record:, title: "Other Page 2")

      sign_in(user_record:)
      visit "/s/#{space_record.identifier}/pages/#{page_record.number}/edit"

      fill_in_editor(text: "[[Page")

      autocomplete_element = find(".cm-tooltip-autocomplete")
      visible_texts = autocomplete_element.find_css(".cm-completionLabel").map(&:visible_text)

      expect(visible_texts).to eq([
        "#{topic_record.name}/Other Page 2",
        "#{topic_record.name}/Other Page 1"
      ])
    end
  end

  describe "リスト記法の自動継続" do
    it "順序なしリスト記法を入力してEnterキーを押すと次の行にもリスト記法が追加されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "- 最初のアイテム")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("- 最初のアイテム\n- ")
    end

    it "順序付きリスト記法を入力してEnterキーを押すと次の行に番号がインクリメントされたリスト記法が追加されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "1. 最初のアイテム")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("1. 最初のアイテム\n2. ")
    end

    it "空のリスト項目でEnterキーを押すとリスト記法が終了すること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "- ")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("")
    end

    it "インデント付きリスト記法でEnterキーを押すとインデントが維持されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "  - インデント付きアイテム")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("  - インデント付きアイテム\n  - ")
    end

    it "異なるマーカー (*、+) でも正常に動作すること" do
      visit_page_editor
      clear_editor
      # * マーカーのテスト
      fill_in_editor(text: "* アスタリスクマーカー")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("* アスタリスクマーカー\n* ")

      # エディターをクリア
      clear_editor

      # + マーカーのテスト
      fill_in_editor(text: "+ プラスマーカー")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("+ プラスマーカー\n+ ")
    end

    it "GitHubタスクリスト記法 (未完了) を入力してEnterキーを押すと次の行にも未完了タスクが追加されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "- [ ] 未完了タスク")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("- [ ] 未完了タスク\n- [ ] ")
    end

    it "GitHubタスクリスト記法 (完了) を入力してEnterキーを押すと次の行に未完了タスクが追加されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "- [x] 完了タスク")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("- [x] 完了タスク\n- [ ] ")
    end

    it "GitHubタスクリスト記法 (完了・大文字X) を入力してEnterキーを押すと次の行に未完了タスクが追加されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "- [X] 完了タスク（大文字）")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("- [X] 完了タスク（大文字）\n- [ ] ")
    end

    it "インデント付きタスクリスト記法でEnterキーを押すとインデントが維持されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "  - [ ] インデント付きタスク")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("  - [ ] インデント付きタスク\n  - [ ] ")
    end

    it "空のタスクリスト項目でEnterキーを押すとタスクリスト記法が終了すること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "- [ ] ")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("")
    end

    it "異なるマーカー (*、+) でもタスクリスト記法が正常に動作すること" do
      visit_page_editor
      clear_editor
      # * マーカーのテスト
      fill_in_editor(text: "* [ ] アスタリスクタスク")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("* [ ] アスタリスクタスク\n* [ ] ")

      # エディターをクリア
      clear_editor

      # + マーカーのテスト
      fill_in_editor(text: "+ [x] プラスタスク")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("+ [x] プラスタスク\n+ [ ] ")
    end

    private def visit_page_editor
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record)
      page_record = create(:page_record, space_record:)
      topic_record = page_record.topic_record
      space_member_record = create(:space_member_record, space_record:, user_record:)
      create(:topic_member_record, space_record:, topic_record:, space_member_record:)

      sign_in(user_record:)
      visit "/s/#{space_record.identifier}/pages/#{page_record.number}/edit"
    end
  end

  private def fill_in_editor(text:)
    within ".cm-content" do
      current_scope.click
      current_scope.send_keys(text)
    end
  end

  private def press_enter_in_editor
    within ".cm-content" do
      current_scope.send_keys(:enter)
    end
  end

  private def get_editor_content
    # CodeMirrorエディターのコンテンツを取得（隠しtextareaから）
    page.evaluate_script("document.querySelector('[data-markdown-editor-target=\"textarea\"]').value")
  end

  private def clear_editor
    # エディターの内容をすべてクリア
    within ".cm-content" do
      current_scope.send_keys([:control, "a"])
      current_scope.send_keys(:delete)
    end
  end
end
