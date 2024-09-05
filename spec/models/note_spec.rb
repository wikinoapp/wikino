# typed: false
# frozen_string_literal: true

RSpec.describe Note, type: :model do
  describe "#titles_in_body" do
    it "returns titles" do
      [
        ["[[a]]", ["a"]],
        ["[[ a ]]", [" a "]],
        ["[[Hello]]", ["Hello"]],
        ["[[こんにちは✌️]]", ["こんにちは✌️"]],
        ["[[a]] [[b]]", ["a", "b"]],
        ["[[Hello]] [[World]]", ["Hello", "World"]],
        ["[[こんにちは]] [[世界🌏]]", ["こんにちは", "世界🌏"]],
        ["[ [a] ]", []],
        ["[[a]", []],
        # A bit weird, but same behavior as Obsidian, Reflect, Bear and etc.
        ["[[[a]]]", ["[a"]],
        ["[[[a]]] [[b]]", ["[a", "b"]],
        ["[[[a]]] [[[b]]]", ["[a", "[b"]],
        ["[[[ a ]]]", ["[ a "]]
      ].each do |(body, expected)|
        note = Note.new(content: NoteContent.new(body:))
        expect(note.titles_in_body).to eq(expected)
      end
    end
  end

  describe "#fetch_links" do
    context "記事にリンクが含まれているとき" do
      let!(:note_1) { create(:note) }
      let!(:note_2) { create(:note) }
      let!(:note_3) { create(:note) }
      let!(:target_note) { create(:note, linked_note_ids: [note_1.id, note_2.id, note_3.id]) }

      it do
        expect(target_note.fetch_links).to eq([note_3, note_2, note_1])
      end
    end
  end
end
